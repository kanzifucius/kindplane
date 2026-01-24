package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	var errs []string

	// Validate cluster config
	if c.Cluster.Name == "" {
		errs = append(errs, "cluster.name is required")
	}

	if c.Cluster.Nodes.ControlPlane < 1 {
		errs = append(errs, "cluster.nodes.controlPlane must be at least 1")
	}

	if c.Cluster.Nodes.Workers < 0 {
		errs = append(errs, "cluster.nodes.workers cannot be negative")
	}

	// Validate port mappings
	for i, pm := range c.Cluster.PortMappings {
		if pm.ContainerPort <= 0 {
			errs = append(errs, fmt.Sprintf("cluster.portMappings[%d].containerPort must be positive", i))
		}
		if pm.HostPort <= 0 {
			errs = append(errs, fmt.Sprintf("cluster.portMappings[%d].hostPort must be positive", i))
		}
		if pm.Protocol != "" && pm.Protocol != "TCP" && pm.Protocol != "UDP" && pm.Protocol != "SCTP" {
			errs = append(errs, fmt.Sprintf("cluster.portMappings[%d].protocol must be TCP, UDP, or SCTP", i))
		}
	}

	// Validate raw config path if provided
	if c.Cluster.RawConfigPath != "" {
		if _, err := os.Stat(c.Cluster.RawConfigPath); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("cluster.rawConfigPath file not found: %s", c.Cluster.RawConfigPath))
		}
	}

	// Validate registry config
	if c.Cluster.Registry.Enabled {
		port := c.Cluster.Registry.GetPort()
		if port < 1 || port > 65535 {
			errs = append(errs, fmt.Sprintf("cluster.registry.port must be between 1 and 65535 (got: %d)", port))
		}
	}

	// Validate trusted CAs config
	registryHosts := make(map[string]bool)
	for i, reg := range c.Cluster.TrustedCAs.Registries {
		if reg.Host == "" {
			errs = append(errs, fmt.Sprintf("cluster.trustedCAs.registries[%d].host is required", i))
		} else {
			// Check for duplicate hosts
			if registryHosts[reg.Host] {
				errs = append(errs, fmt.Sprintf("cluster.trustedCAs.registries[%d].host '%s' is duplicated", i, reg.Host))
			}
			registryHosts[reg.Host] = true
		}
		if reg.CAFile == "" {
			errs = append(errs, fmt.Sprintf("cluster.trustedCAs.registries[%d].caFile is required", i))
		} else {
			if _, err := os.Stat(reg.CAFile); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("cluster.trustedCAs.registries[%d].caFile not found: %s", i, reg.CAFile))
			}
		}
	}

	workloadCANames := make(map[string]bool)
	for i, wl := range c.Cluster.TrustedCAs.Workloads {
		if wl.Name == "" {
			errs = append(errs, fmt.Sprintf("cluster.trustedCAs.workloads[%d].name is required", i))
		} else {
			// Check for duplicate names
			if workloadCANames[wl.Name] {
				errs = append(errs, fmt.Sprintf("cluster.trustedCAs.workloads[%d].name '%s' is duplicated", i, wl.Name))
			}
			workloadCANames[wl.Name] = true
		}
		if wl.CAFile == "" {
			errs = append(errs, fmt.Sprintf("cluster.trustedCAs.workloads[%d].caFile is required", i))
		} else {
			if _, err := os.Stat(wl.CAFile); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("cluster.trustedCAs.workloads[%d].caFile not found: %s", i, wl.CAFile))
			}
		}
	}

	// Validate crossplane config
	if c.Crossplane.Version == "" {
		errs = append(errs, "crossplane.version is required")
	}

	for i, p := range c.Crossplane.Providers {
		if p.Name == "" {
			errs = append(errs, fmt.Sprintf("crossplane.providers[%d].name is required", i))
		}
		if p.Package == "" {
			errs = append(errs, fmt.Sprintf("crossplane.providers[%d].package is required", i))
		}
	}

	// Validate credentials config
	validCredSources := map[string]bool{"env": true, "file": true, "profile": true, "": true}
	if !validCredSources[c.Credentials.AWS.Source] {
		errs = append(errs, fmt.Sprintf("credentials.aws.source must be one of: env, file, profile (got: %s)", c.Credentials.AWS.Source))
	}
	if !validCredSources[c.Credentials.Azure.Source] {
		errs = append(errs, fmt.Sprintf("credentials.azure.source must be one of: env, file, profile (got: %s)", c.Credentials.Azure.Source))
	}

	validK8sSources := map[string]bool{"incluster": true, "kubeconfig": true, "": true}
	if !validK8sSources[c.Credentials.Kubernetes.Source] {
		errs = append(errs, fmt.Sprintf("credentials.kubernetes.source must be one of: incluster, kubeconfig (got: %s)", c.Credentials.Kubernetes.Source))
	}

	// Validate ESO config
	if c.ESO.Enabled && c.ESO.Version == "" {
		errs = append(errs, "eso.version is required when eso.enabled is true")
	}

	// Validate charts config
	validPhases := map[string]bool{
		ChartPhasePrecrossplane:  true,
		ChartPhasePostCrossplane: true,
		ChartPhasePostProviders:  true,
		ChartPhasePostESO:        true,
		"":                       true, // Empty defaults to post-eso
	}
	chartNames := make(map[string]bool)
	for i, chart := range c.Charts {
		if chart.Name == "" {
			errs = append(errs, fmt.Sprintf("charts[%d].name is required", i))
		} else {
			// Check for duplicate release names
			if chartNames[chart.Name] {
				errs = append(errs, fmt.Sprintf("charts[%d].name '%s' is duplicated", i, chart.Name))
			}
			chartNames[chart.Name] = true
		}
		if chart.Repo == "" {
			errs = append(errs, fmt.Sprintf("charts[%d].repo is required", i))
		}
		if chart.Chart == "" {
			errs = append(errs, fmt.Sprintf("charts[%d].chart is required", i))
		}
		if chart.Namespace == "" {
			errs = append(errs, fmt.Sprintf("charts[%d].namespace is required", i))
		}
		if !validPhases[chart.Phase] {
			errs = append(errs, fmt.Sprintf("charts[%d].phase must be one of: pre-crossplane, post-crossplane, post-providers, post-eso (got: %s)", i, chart.Phase))
		}
		// Validate timeout is a valid duration
		if chart.Timeout != "" {
			if _, err := time.ParseDuration(chart.Timeout); err != nil {
				errs = append(errs, fmt.Sprintf("charts[%d].timeout is not a valid duration: %s", i, chart.Timeout))
			}
		}
		// Validate values files exist
		for j, vf := range chart.ValuesFiles {
			if _, err := os.Stat(vf); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("charts[%d].valuesFiles[%d] file not found: %s", i, j, vf))
			}
		}
	}

	// Validate compositions config
	for i, s := range c.Compositions.Sources {
		if s.Type != "local" && s.Type != "git" {
			errs = append(errs, fmt.Sprintf("compositions.sources[%d].type must be 'local' or 'git'", i))
		}
		if s.Type == "local" && s.Path == "" {
			errs = append(errs, fmt.Sprintf("compositions.sources[%d].path is required for local type", i))
		}
		if s.Type == "git" {
			if s.Repo == "" {
				errs = append(errs, fmt.Sprintf("compositions.sources[%d].repo is required for git type", i))
			}
			if s.Branch == "" {
				errs = append(errs, fmt.Sprintf("compositions.sources[%d].branch is required for git type", i))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}
