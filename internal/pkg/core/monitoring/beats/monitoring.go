// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/elastic/elastic-agent/internal/pkg/agent/application/paths"
	monitoringConfig "github.com/elastic/elastic-agent/internal/pkg/core/monitoring/config"
	"github.com/elastic/elastic-agent/pkg/component"
)

const (
	// args: data path, pipeline name, application name
	logFileFormat = "%s/logs/%s/%s"
	// args: data path, install path, pipeline name, application name
	logFileFormatWin = "%s\\logs\\%s\\%s"

	// args: pipeline name, application name
	mbEndpointFileFormatWin = `npipe:///%s-%s`

	// args: pipeline name, application name
	agentMbEndpointFileFormatWin = `npipe:///elastic-agent`
	// agentMbEndpointHTTP is used with cloud and exposes metrics on http endpoint
	agentMbEndpointHTTP = "http://%s:%d"
)

// MonitoringEndpoint is an endpoint where process is exposing its metrics.
func MonitoringEndpoint(spec component.Spec, operatingSystem, pipelineID string) string {
	if endpoint, ok := spec.ProgramSpec.MetricEndpoints[operatingSystem]; ok {
		return endpoint
	}
	if operatingSystem == "windows" {
		return fmt.Sprintf(mbEndpointFileFormatWin, pipelineID, spec.CommandName())
	}
	// unix socket path must be less than 104 characters
	path := fmt.Sprintf("unix://%s.sock", filepath.Join(paths.TempDir(), pipelineID, spec.CommandName(), spec.CommandName()))
	if len(path) < 104 {
		return path
	}
	// place in global /tmp (or /var/tmp on Darwin) to ensure that its small enough to fit; current path is way to long
	// for it to be used, but needs to be unique per Agent (in the case that multiple are running)
	return fmt.Sprintf(`unix:///tmp/elastic-agent/%x.sock`, sha256.Sum256([]byte(path)))
}

func getLoggingFile(spec component.Spec, operatingSystem, installPath, pipelineID string) string {
	if path, ok := spec.ProgramSpec.LogPaths[operatingSystem]; ok {
		return path
	}
	if operatingSystem == "windows" {
		return fmt.Sprintf(logFileFormatWin, paths.Home(), pipelineID, spec.CommandName())
	}
	return fmt.Sprintf(logFileFormat, paths.Home(), pipelineID, spec.CommandName())
}

// AgentMonitoringEndpoint returns endpoint with exposed metrics for agent.
func AgentMonitoringEndpoint(operatingSystem string, cfg *monitoringConfig.MonitoringHTTPConfig) string {
	if cfg != nil && cfg.Enabled {
		return fmt.Sprintf(agentMbEndpointHTTP, cfg.Host, cfg.Port)
	}

	if operatingSystem == "windows" {
		return agentMbEndpointFileFormatWin
	}
	// unix socket path must be less than 104 characters
	path := fmt.Sprintf("unix://%s.sock", filepath.Join(paths.TempDir(), "elastic-agent"))
	if len(path) < 104 {
		return path
	}
	// place in global /tmp to ensure that its small enough to fit; current path is way to long
	// for it to be used, but needs to be unique per Agent (in the case that multiple are running)
	return fmt.Sprintf(`unix:///tmp/elastic-agent/%x.sock`, sha256.Sum256([]byte(path)))
}

// AgentPrefixedMonitoringEndpoint returns endpoint with exposed metrics for agent.
func AgentPrefixedMonitoringEndpoint(operatingSystem string, cfg *monitoringConfig.MonitoringHTTPConfig) string {
	return httpPlusPrefix + AgentMonitoringEndpoint(operatingSystem, cfg)
}
