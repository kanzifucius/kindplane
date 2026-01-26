package doctor

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CheckResult represents the result of a pre-flight check
type CheckResult struct {
	Name        string
	Passed      bool
	Message     string
	Details     string
	Suggestion  string
	Required    bool
}

// Check is a function that performs a pre-flight check
type Check func(ctx context.Context) CheckResult

// RunAllChecks runs all pre-flight checks
func RunAllChecks(ctx context.Context, kubeClient *kubernetes.Clientset) []CheckResult {
	checks := []Check{
		CheckDocker,
		CheckKubectl,
		CheckHelm,
		CheckDiskSpace,
	}

	var results []CheckResult
	for _, check := range checks {
		results = append(results, check(ctx))
	}

	// Add cluster-specific checks if we have a kubeClient
	if kubeClient != nil {
		results = append(results, CheckKubernetesAPI(ctx, kubeClient))
		results = append(results, CheckCrossplaneCRDs(ctx, kubeClient))
	}

	return results
}

// CheckDocker checks if Docker is running and accessible
func CheckDocker(ctx context.Context) CheckResult {
	result := CheckResult{
		Name:     "Docker daemon",
		Required: true,
	}

	// Check if docker command exists
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		result.Passed = false
		result.Message = "Docker not found in PATH"
		result.Suggestion = "Install Docker from https://docs.docker.com/get-docker/"
		return result
	}

	// Check if docker daemon is running
	cmd := exec.CommandContext(ctx, dockerPath, "info", "--format", "{{.ServerVersion}}")
	output, err := cmd.Output()
	if err != nil {
		result.Passed = false
		result.Message = "Docker daemon not running"
		result.Suggestion = "Start Docker daemon: sudo systemctl start docker"
		return result
	}

	version := strings.TrimSpace(string(output))
	result.Passed = true
	result.Message = fmt.Sprintf("Running (v%s)", version)
	return result
}

// CheckKubectl checks if kubectl binary is available
func CheckKubectl(ctx context.Context) CheckResult {
	result := CheckResult{
		Name:     "kubectl binary",
		Required: true,
	}

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		result.Passed = false
		result.Message = "kubectl not found in PATH"
		result.Suggestion = "Install kubectl: https://kubernetes.io/docs/tasks/tools/"
		return result
	}

	cmd := exec.CommandContext(ctx, kubectlPath, "version", "--client", "--short")
	output, err := cmd.Output()
	if err != nil {
		// Try without --short for newer versions
		cmd = exec.CommandContext(ctx, kubectlPath, "version", "--client", "-o", "json")
		output, err = cmd.Output()
		if err != nil {
			result.Passed = false
			result.Message = "Failed to get kubectl version"
			return result
		}
	}

	version := strings.TrimSpace(string(output))
	// Extract version from various output formats
	if strings.Contains(version, "Client Version:") {
		parts := strings.Split(version, ":")
		if len(parts) >= 2 {
			version = strings.TrimSpace(parts[1])
		}
	}
	if len(version) > 30 {
		version = version[:30] + "..."
	}

	result.Passed = true
	result.Message = fmt.Sprintf("Found (%s)", version)
	return result
}

// CheckHelm checks if helm binary is available (optional)
func CheckHelm(ctx context.Context) CheckResult {
	result := CheckResult{
		Name:     "helm binary",
		Required: false,
	}

	helmPath, err := exec.LookPath("helm")
	if err != nil {
		result.Passed = true // Optional, so still passes
		result.Message = "Not found (optional)"
		result.Details = "Helm is bundled in kindplane, but having it installed locally can be useful"
		return result
	}

	cmd := exec.CommandContext(ctx, helmPath, "version", "--short")
	output, err := cmd.Output()
	if err != nil {
		result.Passed = true
		result.Message = "Found but version check failed"
		return result
	}

	version := strings.TrimSpace(string(output))
	result.Passed = true
	result.Message = fmt.Sprintf("Found (%s)", version)
	return result
}

// CheckPort checks if a specific port is available
func CheckPort(port int) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:     fmt.Sprintf("Port %d", port),
			Required: false,
		}

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			result.Passed = false
			result.Message = "In use"
			result.Suggestion = fmt.Sprintf("Run: %s lsof -i :%d", getSudoCommand(), port)
			return result
		}
		listener.Close()

		result.Passed = true
		result.Message = "Available"
		return result
	}
}

// CheckKubernetesAPI checks if the Kubernetes API is accessible
func CheckKubernetesAPI(ctx context.Context, kubeClient *kubernetes.Clientset) CheckResult {
	result := CheckResult{
		Name:     "Kubernetes API",
		Required: false,
	}

	if kubeClient == nil {
		result.Passed = true
		result.Message = "No cluster connected"
		return result
	}

	_, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		result.Passed = false
		result.Message = "Unreachable"
		result.Details = err.Error()
		return result
	}

	result.Passed = true
	result.Message = "Reachable"
	return result
}

// CheckCrossplaneCRDs checks if Crossplane CRDs are installed
func CheckCrossplaneCRDs(ctx context.Context, kubeClient *kubernetes.Clientset) CheckResult {
	result := CheckResult{
		Name:     "Crossplane CRDs",
		Required: false,
	}

	if kubeClient == nil {
		result.Passed = true
		result.Message = "No cluster connected"
		return result
	}

	// Check for crossplane-system namespace as a proxy for CRDs
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, "crossplane-system", metav1.GetOptions{})
	if err != nil {
		result.Passed = true
		result.Message = "Not installed"
		result.Details = "Run 'kindplane up' to install Crossplane"
		return result
	}

	result.Passed = true
	result.Message = "Installed"
	return result
}

func getSudoCommand() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	return "sudo"
}
