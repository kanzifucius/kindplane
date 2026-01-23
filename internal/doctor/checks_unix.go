//go:build linux || darwin

package doctor

import (
	"context"
	"fmt"
	"os"
	"syscall"
)

// CheckDiskSpace checks if there's sufficient disk space
func CheckDiskSpace(ctx context.Context) CheckResult {
	result := CheckResult{
		Name:     "Disk space",
		Required: true,
	}

	// Get current working directory's filesystem stats
	var stat syscall.Statfs_t
	wd, _ := os.Getwd()
	err := syscall.Statfs(wd, &stat)
	if err != nil {
		result.Passed = true
		result.Message = "Unable to check (skipped)"
		return result
	}

	// Calculate available space in GB
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	availableGB := float64(availableBytes) / (1024 * 1024 * 1024)

	// Require at least 5GB for a Kind cluster
	minRequired := 5.0
	if availableGB < minRequired {
		result.Passed = false
		result.Message = fmt.Sprintf("%.1f GB available (%.1f GB required)", availableGB, minRequired)
		result.Suggestion = "Free up disk space before creating a cluster"
		return result
	}

	result.Passed = true
	result.Message = fmt.Sprintf("%.1f GB available", availableGB)
	return result
}
