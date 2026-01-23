//go:build windows

package doctor

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// CheckDiskSpace checks if there's sufficient disk space
func CheckDiskSpace(ctx context.Context) CheckResult {
	result := CheckResult{
		Name:     "Disk space",
		Required: true,
	}

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		result.Passed = true
		result.Message = "Unable to check (skipped)"
		return result
	}

	// Use Windows API to get disk free space
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	pathPtr, err := syscall.UTF16PtrFromString(wd)
	if err != nil {
		result.Passed = true
		result.Message = "Unable to check (skipped)"
		return result
	}

	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		result.Passed = true
		result.Message = "Unable to check (skipped)"
		return result
	}

	// Calculate available space in GB
	availableGB := float64(freeBytesAvailable) / (1024 * 1024 * 1024)

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
