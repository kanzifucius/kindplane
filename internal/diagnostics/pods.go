package diagnostics

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodDiagnostic contains diagnostic information about a pod
type PodDiagnostic struct {
	Name            string
	Namespace       string
	Phase           string
	Ready           bool
	ReadyContainers int
	TotalContainers int
	Conditions      []PodCondition
	Containers      []ContainerDiagnostic
	Error           string
}

// PodCondition represents a pod condition
type PodCondition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// ContainerDiagnostic contains diagnostic information about a container
type ContainerDiagnostic struct {
	Name              string
	Ready             bool
	State             string
	Restarts          int32
	WaitingReason     string
	WaitingMessage    string
	TerminatedReason  string
	TerminatedMessage string
	ExitCode          int32
	RecentLogs        []string
}

// HasIssue returns true if the container has any issues
func (c *ContainerDiagnostic) HasIssue() bool {
	if !c.Ready {
		return true
	}
	if c.Restarts > 0 {
		return true
	}
	if c.WaitingReason != "" && c.WaitingReason != "ContainerCreating" {
		return true
	}
	if c.TerminatedReason != "" && c.TerminatedReason != "Completed" {
		return true
	}
	if c.ExitCode != 0 {
		return true
	}
	return false
}

// IsCrashLooping returns true if the container is in CrashLoopBackOff
func (c *ContainerDiagnostic) IsCrashLooping() bool {
	return c.WaitingReason == "CrashLoopBackOff"
}

// IsImagePullError returns true if the container has image pull issues
func (c *ContainerDiagnostic) IsImagePullError() bool {
	return c.WaitingReason == "ImagePullBackOff" ||
		c.WaitingReason == "ErrImagePull" ||
		c.WaitingReason == "ErrImageNeverPull"
}

// CollectPodDiagnostics collects diagnostics for all pods in the given context
func (c *Collector) CollectPodDiagnostics(ctx context.Context, diagCtx Context) ([]PodDiagnostic, error) {
	listOpts := metav1.ListOptions{}
	if diagCtx.LabelSelector != "" {
		listOpts.LabelSelector = diagCtx.LabelSelector
	}

	pods, err := c.kubeClient.CoreV1().Pods(diagCtx.Namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	var diagnostics []PodDiagnostic
	for _, pod := range pods.Items {
		diag := c.collectPodDiagnostic(ctx, &pod, diagCtx)
		diagnostics = append(diagnostics, diag)
	}

	return diagnostics, nil
}

// collectPodDiagnostic collects diagnostic information for a single pod
func (c *Collector) collectPodDiagnostic(ctx context.Context, pod *corev1.Pod, diagCtx Context) PodDiagnostic {
	diag := PodDiagnostic{
		Name:            pod.Name,
		Namespace:       pod.Namespace,
		Phase:           string(pod.Status.Phase),
		TotalContainers: len(pod.Spec.Containers),
	}

	// Collect conditions
	for _, cond := range pod.Status.Conditions {
		diag.Conditions = append(diag.Conditions, PodCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})

		// Check if pod is ready
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			diag.Ready = true
		}
	}

	// Collect container statuses
	containerStatuses := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
	for _, cs := range containerStatuses {
		containerDiag := collectContainerDiagnostic(cs)

		// Check if container is ready
		if cs.Ready {
			diag.ReadyContainers++
		}

		// Collect logs for containers with issues
		if containerDiag.HasIssue() && diagCtx.MaxLogLines > 0 {
			logs, err := c.getContainerLogs(ctx, pod.Namespace, pod.Name, cs.Name, diagCtx.MaxLogLines)
			if err == nil {
				containerDiag.RecentLogs = logs
			}
		}

		diag.Containers = append(diag.Containers, containerDiag)
	}

	return diag
}

// collectContainerDiagnostic extracts diagnostic information from a container status
func collectContainerDiagnostic(cs corev1.ContainerStatus) ContainerDiagnostic {
	diag := ContainerDiagnostic{
		Name:     cs.Name,
		Ready:    cs.Ready,
		Restarts: cs.RestartCount,
	}

	// Determine state
	if cs.State.Running != nil {
		diag.State = "Running"
	} else if cs.State.Waiting != nil {
		diag.State = "Waiting"
		diag.WaitingReason = cs.State.Waiting.Reason
		diag.WaitingMessage = cs.State.Waiting.Message
	} else if cs.State.Terminated != nil {
		diag.State = "Terminated"
		diag.TerminatedReason = cs.State.Terminated.Reason
		diag.TerminatedMessage = cs.State.Terminated.Message
		diag.ExitCode = cs.State.Terminated.ExitCode
	}

	// Check last terminated state if container is waiting (for restart info)
	if cs.LastTerminationState.Terminated != nil {
		last := cs.LastTerminationState.Terminated
		if diag.TerminatedReason == "" {
			diag.TerminatedReason = last.Reason
			diag.TerminatedMessage = last.Message
			diag.ExitCode = last.ExitCode
		}
	}

	return diag
}

// getContainerLogs fetches the most recent logs from a container
func (c *Collector) getContainerLogs(ctx context.Context, namespace, podName, containerName string, lines int64) ([]string, error) {
	opts := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &lines,
	}

	req := c.kubeClient.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("opening log stream: %w", err)
	}
	defer stream.Close()

	var logLines []string
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		// Clean up the line
		line = strings.TrimSpace(line)
		if line != "" {
			logLines = append(logLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return logLines, fmt.Errorf("reading logs: %w", err)
	}

	return logLines, nil
}

// CollectPodDiagnosticsForDeployment collects diagnostics for pods belonging to a deployment
func (c *Collector) CollectPodDiagnosticsForDeployment(ctx context.Context, namespace, deploymentName string, maxLogLines int64) ([]PodDiagnostic, error) {
	// Get the deployment to find the label selector
	deployment, err := c.kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting deployment: %w", err)
	}

	// Build label selector from deployment spec
	var selectors []string
	for key, value := range deployment.Spec.Selector.MatchLabels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", key, value))
	}

	diagCtx := Context{
		Namespace:     namespace,
		LabelSelector: strings.Join(selectors, ","),
		MaxLogLines:   maxLogLines,
	}

	return c.CollectPodDiagnostics(ctx, diagCtx)
}
