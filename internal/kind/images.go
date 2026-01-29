package kind

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	"github.com/kanzi/kindplane/internal/config"
)

// PreloadResult contains the result of image pre-loading
type PreloadResult struct {
	LoadedCount   int      // Number of images successfully loaded
	MissingImages []string // Images that were expected but not found locally
	RePulledCount int      // Number of images re-pulled for correct architecture
}

// imageArchInfo holds architecture information for an image
type imageArchInfo struct {
	Architecture string
	OS           string
	Variant      string // e.g., "v8" for arm64
}

// getNodeArchitecture returns the architecture of the Kind node
func getNodeArchitecture(ctx context.Context, clusterName string) (string, error) {
	provider := cluster.NewProvider()
	nodes, err := provider.ListNodes(clusterName)
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}
	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found in cluster %s", clusterName)
	}

	// Get architecture from first node using uname -m
	cmd := exec.CommandContext(ctx, "docker", "exec", nodes[0].String(), "uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get node architecture: %w", err)
	}

	arch := strings.TrimSpace(string(output))
	// Normalize architecture names (uname returns x86_64, but Docker uses amd64)
	return normalizeArch(arch), nil
}

// normalizeArch converts system architecture names to Docker/OCI format
func normalizeArch(arch string) string {
	switch arch {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		return arch
	}
}

// getImageArchitecture returns the architecture of a local Docker image
func getImageArchitecture(ctx context.Context, imageName string) (*imageArchInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", imageName, "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	var inspectResult struct {
		Architecture string `json:"Architecture"`
		Os           string `json:"Os"`
		Variant      string `json:"Variant"`
	}

	if err := json.Unmarshal(output, &inspectResult); err != nil {
		return nil, fmt.Errorf("failed to parse image inspect output: %w", err)
	}

	return &imageArchInfo{
		Architecture: inspectResult.Architecture,
		OS:           inspectResult.Os,
		Variant:      inspectResult.Variant,
	}, nil
}

// pullImageForPlatform pulls an image for a specific platform
func pullImageForPlatform(ctx context.Context, imageName, platform string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", "--platform", platform, imageName)
	return cmd.Run()
}

// getTargetPlatform returns the platform string for pulling images (e.g., "linux/amd64")
func getTargetPlatform(arch string) string {
	return fmt.Sprintf("linux/%s", arch)
}

// isArchitectureMismatch checks if the image architecture matches the target
func isArchitectureMismatch(imageArch, targetArch string) bool {
	// Normalize both for comparison
	imageArch = normalizeArch(imageArch)
	targetArch = normalizeArch(targetArch)
	return imageArch != targetArch
}

// getHostArchitecture returns the architecture of the host machine
func getHostArchitecture() string {
	return normalizeArch(runtime.GOARCH)
}

// cmdReader wraps a ReadCloser and ensures the command is properly closed
type cmdReader struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (r *cmdReader) Close() error {
	r.ReadCloser.Close()
	return r.cmd.Wait()
}

// PullImages pulls Docker images from remote registries
func PullImages(ctx context.Context, images []string, logFn func(string)) (int, error) {
	successCount := 0
	for i, img := range images {
		logFn(fmt.Sprintf("Pulling %s (%d/%d)...", getShortImageName(img), i+1, len(images)))

		cmd := exec.CommandContext(ctx, "docker", "pull", img)
		if err := cmd.Run(); err != nil {
			logFn(fmt.Sprintf("Warning: Failed to pull %s: %v", getShortImageName(img), err))
			continue
		}

		logFn(fmt.Sprintf("✓ Pulled %s", getShortImageName(img)))
		successCount++
	}
	return successCount, nil
}

// PreloadImages pre-loads images from local Docker into Kind cluster
// Supports two modes:
//   - Registry mode: Push images to local registry (if cfg.Cluster.Registry.Enabled)
//   - Direct mode: Load images directly into Kind nodes
func PreloadImages(ctx context.Context, clusterName string, cfg *config.Config, logFn func(string)) (*PreloadResult, error) {
	result := &PreloadResult{
		LoadedCount:   0,
		MissingImages: []string{},
		RePulledCount: 0,
	}

	// Check if caching is enabled
	if cfg.Crossplane.ImageCache != nil && !cfg.Crossplane.ImageCache.IsEnabled() {
		return result, nil
	}

	// Collect all images to preload
	images := collectImagesToPreload(cfg)
	if len(images) == 0 {
		return result, nil
	}

	// Detect Kind node architecture
	nodeArch, err := getNodeArchitecture(ctx, clusterName)
	if err != nil {
		logFn(fmt.Sprintf("Warning: Could not detect node architecture: %v", err))
		// Fall back to host architecture
		nodeArch = getHostArchitecture()
	}
	logFn(fmt.Sprintf("Target architecture: %s", nodeArch))

	// Filter to only locally available images and check architecture
	localImages, rePulledCount, err := filterAndFixLocalImages(ctx, images, nodeArch, logFn)
	if err != nil {
		return result, err
	}
	result.RePulledCount = rePulledCount

	// Track missing images
	localImageSet := make(map[string]bool)
	for _, img := range localImages {
		localImageSet[img] = true
	}
	for _, img := range images {
		if !localImageSet[img] {
			result.MissingImages = append(result.MissingImages, img)
		}
	}

	if len(localImages) == 0 {
		logFn("No images found locally (will pull from registry)")
		return result, nil
	}

	logFn(fmt.Sprintf("Found %d/%d images locally", len(localImages), len(images)))

	// Choose mode based on registry configuration
	var loadErr error
	if cfg.Cluster.Registry.Enabled {
		// Registry mode: Push to local registry
		registryHost := fmt.Sprintf("localhost:%d", cfg.Cluster.Registry.GetPort())
		loadErr = pushImagesToRegistry(ctx, localImages, registryHost, logFn)
	} else {
		// Direct mode: Load into Kind nodes
		loadErr = loadImagesIntoNodes(ctx, clusterName, localImages, logFn)
	}

	if loadErr == nil {
		result.LoadedCount = len(localImages)
	}

	return result, loadErr
}

// collectImagesToPreload gathers all images that should be pre-loaded
func collectImagesToPreload(cfg *config.Config) []string {
	images := make(map[string]bool) // Use map to deduplicate

	// 1. Crossplane core images
	if cfg.Crossplane.ImageCache == nil || cfg.Crossplane.ImageCache.ShouldPreloadCrossplane() {
		for _, img := range deriveCrossplaneImages(cfg.Crossplane.Version) {
			images[img] = true
		}
	}

	// 2. Provider images
	if cfg.Crossplane.ImageCache == nil || cfg.Crossplane.ImageCache.ShouldPreloadProviders() {
		for _, provider := range cfg.Crossplane.Providers {
			// Check for overrides first
			if cfg.Crossplane.ImageCache != nil {
				if overrides, ok := cfg.Crossplane.ImageCache.ImageOverrides[provider.Package]; ok {
					for _, img := range overrides {
						images[img] = true
					}
					continue
				}
			}

			// Use convention-based derivation
			for _, img := range deriveProviderImages(provider.Package) {
				images[img] = true
			}
		}
	}

	// 3. Additional images
	if cfg.Crossplane.ImageCache != nil {
		for _, img := range cfg.Crossplane.ImageCache.AdditionalImages {
			images[img] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(images))
	for img := range images {
		result = append(result, img)
	}

	return result
}

// deriveProviderImages derives controller image names from provider packages
// Convention: xpkg.upbound.io/publisher/provider:version
//
//	-> xpkg.upbound.io/publisher/provider:version (the xpkg package)
//	-> xpkg.upbound.io/publisher/provider-controller:version (controller)
func deriveProviderImages(pkg string) []string {
	// Parse package: registry/publisher/package:tag
	parts := strings.Split(pkg, "/")
	if len(parts) < 3 {
		return []string{pkg} // Return as-is if format unexpected
	}

	registry := parts[0]
	publisher := parts[1]
	packageWithTag := parts[2]

	// Split package from tag
	pkgParts := strings.SplitN(packageWithTag, ":", 2)
	if len(pkgParts) != 2 {
		return []string{pkg}
	}

	packageName := pkgParts[0]
	tag := pkgParts[1]

	// Derive images
	images := []string{
		pkg, // The xpkg package itself
		fmt.Sprintf("%s/%s/%s-controller:%s", registry, publisher, packageName, tag), // Controller image
	}

	return images
}

// deriveCrossplaneImages derives Crossplane core images from version
// Returns: crossplane/crossplane:v{version}, crossplane/crossplane-rbac-manager:v{version}
func deriveCrossplaneImages(version string) []string {
	// Ensure version has 'v' prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Standard Crossplane images from official registry
	return []string{
		fmt.Sprintf("crossplane/crossplane:%s", version),
		fmt.Sprintf("crossplane/crossplane-rbac-manager:%s", version),
	}
}

// filterAndFixLocalImages checks which images exist locally and have correct architecture.
// If an image exists but has wrong architecture, it attempts to re-pull with correct platform.
// Returns: list of valid local images, count of re-pulled images, error
func filterAndFixLocalImages(ctx context.Context, images []string, targetArch string, logFn func(string)) ([]string, int, error) {
	var localImages []string
	rePulledCount := 0
	targetPlatform := getTargetPlatform(targetArch)

	for _, img := range images {
		exists, err := imageExistsLocally(ctx, img)
		if err != nil {
			// Don't fail on individual image checks, just skip
			continue
		}

		if !exists {
			// Image not available locally
			continue
		}

		// Check image architecture
		archInfo, err := getImageArchitecture(ctx, img)
		if err != nil {
			logFn(fmt.Sprintf("Warning: Could not check architecture for %s: %v", getShortImageName(img), err))
			// Include it anyway and let the load fail if there's an issue
			localImages = append(localImages, img)
			continue
		}

		if isArchitectureMismatch(archInfo.Architecture, targetArch) {
			logFn(fmt.Sprintf("Image %s has wrong architecture (%s, need %s), re-pulling...",
				getShortImageName(img), archInfo.Architecture, targetArch))

			// Try to re-pull with correct platform
			if err := pullImageForPlatform(ctx, img, targetPlatform); err != nil {
				logFn(fmt.Sprintf("Warning: Failed to re-pull %s for %s: %v",
					getShortImageName(img), targetPlatform, err))
				continue
			}

			logFn(fmt.Sprintf("✓ Re-pulled %s for %s", getShortImageName(img), targetArch))
			rePulledCount++
		}

		localImages = append(localImages, img)
	}

	return localImages, rePulledCount, nil
}

// imageExistsLocally checks if an image exists in local Docker daemon
func imageExistsLocally(ctx context.Context, imageName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", imageName)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// pushImagesToRegistry pushes local images to the local registry
func pushImagesToRegistry(ctx context.Context, images []string, registryHost string, logFn func(string)) error {
	successCount := 0

	for _, img := range images {
		// Tag image for local registry
		registryImage := retagForRegistry(img, registryHost)

		// Tag command
		tagCmd := exec.CommandContext(ctx, "docker", "tag", img, registryImage)
		if err := tagCmd.Run(); err != nil {
			logFn(fmt.Sprintf("Warning: Failed to tag %s: %v", img, err))
			continue
		}

		// Push to registry
		pushCmd := exec.CommandContext(ctx, "docker", "push", registryImage)
		// Suppress output to avoid clutter
		if err := pushCmd.Run(); err != nil {
			logFn(fmt.Sprintf("Warning: Failed to push %s: %v", registryImage, err))
			continue
		}

		logFn(fmt.Sprintf("✓ Cached %s in local registry", getShortImageName(img)))
		successCount++
	}

	if successCount > 0 {
		logFn(fmt.Sprintf("Pre-loaded %d images to registry", successCount))
	}

	return nil
}

// retagForRegistry retags an image for the local registry
func retagForRegistry(imageName, registryHost string) string {
	// Remove any existing registry prefix and add local registry
	// e.g., xpkg.upbound.io/upbound/provider-aws:v1 -> localhost:5001/upbound/provider-aws:v1
	// But keep Docker Hub images intact: crossplane/crossplane:v1 -> localhost:5001/crossplane/crossplane:v1

	parts := strings.SplitN(imageName, "/", 2)
	if len(parts) == 2 {
		// Check if first part looks like a registry (contains a dot)
		if strings.Contains(parts[0], ".") {
			// Has registry prefix, replace it
			return fmt.Sprintf("%s/%s", registryHost, parts[1])
		}
	}
	// No registry prefix or Docker Hub image (e.g., crossplane/crossplane:v1.15.0)
	return fmt.Sprintf("%s/%s", registryHost, imageName)
}

// loadImagesIntoNodes loads images directly into Kind nodes using Kind SDK
func loadImagesIntoNodes(ctx context.Context, clusterName string, images []string, logFn func(string)) error {
	// Get Kind provider
	provider := cluster.NewProvider()

	// List all nodes in the cluster
	nodes, err := provider.ListNodes(clusterName)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found in cluster %s", clusterName)
	}

	logFn(fmt.Sprintf("Loading images into %d node(s)", len(nodes)))

	successCount := 0

	// Load each image
	for _, img := range images {
		logFn(fmt.Sprintf("Processing %s...", getShortImageName(img)))

		// Check if image is already loaded in the first node
		if _, err := nodeutils.ImageID(nodes[0], img); err == nil {
			logFn(fmt.Sprintf("  ✓ Already present"))
			successCount++
			continue
		}

		// For macOS with Docker Desktop, we need to handle each node separately
		// and create a fresh stream for each node
		allNodesSuccess := true

		// Use a temporary file to transfer the image to avoid pipe issues
		// This is a workaround for some environments where piping directly to ctr fails

		// 1. Create a temporary file
		tmpFile, err := os.CreateTemp("", "kind-image-*.tar")
		if err != nil {
			logFn(fmt.Sprintf("  ✗ Failed to create temp file: %v", err))
			continue
		}
		defer os.Remove(tmpFile.Name()) // Clean up

		// 2. Save image to the temporary file
		logFn(fmt.Sprintf("  Exporting image to temp file..."))
		saveCmd := exec.CommandContext(ctx, "docker", "save", "-o", tmpFile.Name(), img)
		if out, err := saveCmd.CombinedOutput(); err != nil {
			logFn(fmt.Sprintf("  ✗ Failed to save image: %v (%s)", err, strings.TrimSpace(string(out))))
			tmpFile.Close()
			continue
		}
		tmpFile.Close() // Close so we can read it again

		// WORKAROUND: Remove OCI index files from the tarball if present.
		// Docker on macOS (ARM64) sometimes exports a tarball containing both OCI layout/index.json AND
		// a Docker v2 manifest.json. However, the 'index.json' often references a multi-arch manifest
		// digest (e.g., from the remote registry) that does not exist in the local blob store because
		// we only pulled the single architecture.
		//
		// When 'ctr' sees 'index.json', it tries to validate the entire index (all architectures),
		// fails to find the other architecture blobs, and errors with "content digest ... not found".
		//
		// By removing 'index.json' and 'oci-layout', we force 'ctr' (and 'kind load') to fall back
		// to using 'manifest.json' (Docker V2 format), which correctly points to the single-arch
		// blobs that are actually present in the tarball.
		if runtime.GOARCH == "arm64" {
			// Create a temporary directory for repackaging
			repackDir, err := os.MkdirTemp("", "kind-repack-*")
			if err == nil {
				defer os.RemoveAll(repackDir)

				// Unpack tarball
				unpackCmd := exec.CommandContext(ctx, "tar", "-xf", tmpFile.Name(), "-C", repackDir)
				if err := unpackCmd.Run(); err == nil {
					// Check if index.json exists
					if _, err := os.Stat(fmt.Sprintf("%s/index.json", repackDir)); err == nil {
						// Remove OCI files
						os.Remove(fmt.Sprintf("%s/index.json", repackDir))
						os.Remove(fmt.Sprintf("%s/oci-layout", repackDir))

						// Repack into the temp file (overwriting it)
						// Note: We need to recreate the file
						tmpFileForRepack, err := os.Create(tmpFile.Name())
						if err == nil {
							// Tar command to repack relative to repackDir
							// tar -C repackDir -cf tmpFile.Name .
							repackCmd := exec.CommandContext(ctx, "tar", "-C", repackDir, "-cf", tmpFileForRepack.Name(), ".")
							if err := repackCmd.Run(); err != nil {
								logFn(fmt.Sprintf("  Warning: Failed to repack image (continuing with original): %v", err))
							}
							tmpFileForRepack.Close()
						}
					}
				}
			}
		}

		// 3. Load via `kind load image-archive` command directly instead of using the library
		// The library implementation seems to have issues with pipes on some environments
		// and we are already shelling out for docker commands anyway.

		logFn(fmt.Sprintf("  Loading archive into cluster nodes..."))
		loadCmd := exec.CommandContext(ctx, "kind", "load", "image-archive", tmpFile.Name(), "--name", clusterName)
		out, err := loadCmd.CombinedOutput()
		_ = out // Prevent unused variable error if we don't use it in success path
		if err != nil {
			logFn(fmt.Sprintf("  Warning: Standard load failed, trying legacy import: %v", err))

			// Fallback: Manually copy and import into each node
			// This works around issues where `kind load` or streaming fails
			allNodesSuccess = true // assume success unless one node fails

			for _, n := range nodes {
				// 1. Copy file to node container
				containerName := n.String()
				// Use a simpler filename to avoid potential character issues in shell
				// Copy to root instead of /tmp because /tmp behaves weirdly in some Kind nodes/Docker Desktop setups
				simpleName := fmt.Sprintf("image-%d.tar", time.Now().UnixNano())
				targetPath := fmt.Sprintf("/%s", simpleName)

				logFn(fmt.Sprintf("  Copying to %s:%s...", containerName, targetPath))
				cpCmd := exec.CommandContext(ctx, "docker", "cp", tmpFile.Name(), fmt.Sprintf("%s:%s", containerName, targetPath))
				if out, err := cpCmd.CombinedOutput(); err != nil {
					logFn(fmt.Sprintf("  ✗ Failed to copy to node %s: %v (%s)", containerName, err, strings.TrimSpace(string(out))))
					allNodesSuccess = false
					continue
				}

				// 2. Import using ctr inside the node
				logFn(fmt.Sprintf("  Importing in %s...", containerName))
				// Try with --no-unpack first as it's often more robust
				// NOTE: We need to use the path relative to the container filesystem.
				// The previous copy put it at /tmp/... inside the container.
				importCmd := exec.CommandContext(ctx, "docker", "exec", containerName, "ctr", "-n", "k8s.io", "images", "import", "--no-unpack", targetPath)
				out, err := importCmd.CombinedOutput()
				_ = out // Prevent unused variable error

				if err != nil {
					// Retry without --no-unpack
					importCmd = exec.CommandContext(ctx, "docker", "exec", containerName, "ctr", "-n", "k8s.io", "images", "import", targetPath)
					if out2, err2 := importCmd.CombinedOutput(); err2 != nil {
						logFn(fmt.Sprintf("  ✗ Failed to import in node %s: %v\nOutput 1: %s\nOutput 2: %s",
							containerName, err2, strings.TrimSpace(string(out)), strings.TrimSpace(string(out2))))
						allNodesSuccess = false
					}
				}

				// Cleanup temp file in node
				exec.CommandContext(ctx, "docker", "exec", containerName, "rm", targetPath).Run()
			}
		}

		if allNodesSuccess {
			logFn(fmt.Sprintf("  ✓ Loaded successfully"))
			successCount++
		}
	}

	if successCount > 0 {
		logFn(fmt.Sprintf("Pre-loaded %d/%d images into Kind nodes", successCount, len(images)))
	} else {
		logFn("No images were successfully loaded")
	}

	return nil
}

// saveDockerImage exports a Docker image as a tar stream
func saveDockerImage(ctx context.Context, imageName string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "docker", "save", imageName)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start docker save: %w", err)
	}

	// Return reader that closes the command when done
	return &cmdReader{ReadCloser: stdout, cmd: cmd}, nil
}

// getShortImageName returns a shortened version of the image name for display
func getShortImageName(imageName string) string {
	// For xpkg images, show just publisher/package:tag
	// e.g., xpkg.upbound.io/upbound/provider-aws:v1 -> upbound/provider-aws:v1
	parts := strings.Split(imageName, "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		// Has registry prefix, return last two parts
		return fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1])
	}
	return imageName
}
