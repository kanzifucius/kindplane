package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	logsComponent  string
	logsFollow     bool
	logsTail       int64
	logsSince      time.Duration
	logsContainer  string
	logsPrevious   bool
	logsTimestamps bool
	logsTimeout    time.Duration
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream logs from cluster components",
	Long: `Stream logs from Crossplane and related components in the cluster.

By default, this command streams logs from the Crossplane controller.
You can specify different components or pod names to view their logs.`,
	Example: `  # View Crossplane controller logs
  kindplane logs

  # View logs for a specific component
  kindplane logs --component providers

  # Follow logs in real-time
  kindplane logs --follow

  # View last 100 lines
  kindplane logs --tail 100

  # View logs from the last 5 minutes
  kindplane logs --since 5m

  # View logs from a specific pod
  kindplane logs --component crossplane-rbac-manager

  # View previous container logs (after restart)
  kindplane logs --previous`,
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().StringVar(&logsComponent, "component", "crossplane", "Component to view logs for (crossplane, providers, eso, or pod name)")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().Int64Var(&logsTail, "tail", 100, "Number of lines to show from the end of the logs")
	logsCmd.Flags().DurationVar(&logsSince, "since", 0, "Only show logs newer than a relative duration (e.g., 5m, 1h)")
	logsCmd.Flags().StringVar(&logsContainer, "container", "", "Container name (if pod has multiple containers)")
	logsCmd.Flags().BoolVar(&logsPrevious, "previous", false, "Show previous terminated container logs")
	logsCmd.Flags().BoolVar(&logsTimestamps, "timestamps", false, "Include timestamps in log output")
	logsCmd.Flags().DurationVar(&logsTimeout, "timeout", 5*time.Minute, "Timeout for log streaming")
}

func runLogs(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), logsTimeout)
	defer cancel()

	clusterName := cfg.Cluster.Name

	// Check if cluster exists
	exists, err := kind.ClusterExists(clusterName)
	if err != nil {
		printError("Failed to check cluster status: %v", err)
		return err
	}

	if !exists {
		printError("Cluster '%s' does not exist. Run 'kindplane up' first.", clusterName)
		return fmt.Errorf("cluster not found")
	}

	// Get kubernetes client
	kubeClient, err := kind.GetKubeClient(clusterName)
	if err != nil {
		printError("Failed to connect to cluster: %v", err)
		return err
	}

	// Determine namespace and label selector based on component
	var namespace, labelSelector, podName string
	switch logsComponent {
	case "crossplane":
		namespace = "crossplane-system"
		labelSelector = "app=crossplane"
	case "providers":
		namespace = "crossplane-system"
		labelSelector = "pkg.crossplane.io/provider"
	case "eso":
		namespace = "external-secrets"
		labelSelector = "app.kubernetes.io/name=external-secrets"
	default:
		// Treat as pod name - search in crossplane-system first
		namespace = "crossplane-system"
		podName = logsComponent
	}

	// Find pods
	var pods *corev1.PodList
	if podName != "" {
		// Get specific pod
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			// Try in other namespaces
			for _, ns := range []string{"external-secrets", "default", "kube-system"} {
				pod, err = kubeClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
				if err == nil {
					namespace = ns
					break
				}
			}
			if err != nil {
				printError("Pod '%s' not found", podName)
				return err
			}
		}
		pods = &corev1.PodList{Items: []corev1.Pod{*pod}}
	} else {
		pods, err = kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			printError("Failed to list pods: %v", err)
			return err
		}
	}

	if len(pods.Items) == 0 {
		printWarn("No pods found for component: %s", logsComponent)
		return nil
	}

	// Stream logs from the first running pod
	var targetPod *corev1.Pod
	for i := range pods.Items {
		if pods.Items[i].Status.Phase == corev1.PodRunning {
			targetPod = &pods.Items[i]
			break
		}
	}
	if targetPod == nil {
		targetPod = &pods.Items[0]
	}

	fmt.Println()
	fmt.Println(ui.Title(ui.IconFile + " Logs"))
	fmt.Println(ui.Divider())
	printInfo("Streaming logs from pod: %s/%s", targetPod.Namespace, targetPod.Name)
	fmt.Println()

	// Build log options
	logOptions := &corev1.PodLogOptions{
		Follow:     logsFollow,
		Previous:   logsPrevious,
		Timestamps: logsTimestamps,
	}

	if logsTail > 0 {
		logOptions.TailLines = &logsTail
	}

	if logsSince > 0 {
		sinceSeconds := int64(logsSince.Seconds())
		logOptions.SinceSeconds = &sinceSeconds
	}

	if logsContainer != "" {
		logOptions.Container = logsContainer
	} else if len(targetPod.Spec.Containers) > 0 {
		// Default to first container
		logOptions.Container = targetPod.Spec.Containers[0].Name
	}

	// Get log stream
	req := kubeClient.CoreV1().Pods(targetPod.Namespace).GetLogs(targetPod.Name, logOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		printError("Failed to stream logs: %v", err)
		return err
	}
	defer stream.Close()

	// Stream logs to stdout
	reader := bufio.NewReader(stream)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			if ctx.Err() != nil {
				break
			}
			return err
		}
		fmt.Print(line)
	}

	return nil
}
