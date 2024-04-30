// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/elastic/elastic-agent/internal/pkg/release"
)

const buildDescription = "Elastic opentelemetry-collector distribution"

func Run(ctx context.Context, stop chan bool, configFiles []string) error {
	fmt.Fprintln(os.Stdout, "Starting in otel mode")
	settings, err := newSettings(release.Version(), configFiles)
	if err != nil {
		return err
	}

	svc, err := otelcol.NewCollector(*settings)
	if err != nil {
		return err
	}

	// cancel context on stop from event manager
	cancelCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-stop
		cancel()
	}()
	defer cancel()

	return svc.Run(cancelCtx)
}

func newSettings(version string, configPaths []string) (*otelcol.CollectorSettings, error) {
	buildInfo := component.BuildInfo{
		Command:     os.Args[0],
		Description: buildDescription,
		Version:     version,
	}
	converterSet := confmap.ConverterSettings{}
	configProviderSettings := otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: configPaths,
			Providers: makeMapProvidersMap(
				fileprovider.NewWithSettings(confmap.ProviderSettings{}),
				envprovider.NewWithSettings(confmap.ProviderSettings{}),
				yamlprovider.NewWithSettings(confmap.ProviderSettings{}),
				httpprovider.NewWithSettings(confmap.ProviderSettings{}),
				httpsprovider.NewWithSettings(confmap.ProviderSettings{}),
			),
			Converters: []confmap.Converter{expandconverter.New(converterSet)},
		},
	}
	provider, err := otelcol.NewConfigProvider(configProviderSettings)
	if err != nil {
		return nil, err
	}

	return &otelcol.CollectorSettings{
		Factories:      components,
		BuildInfo:      buildInfo,
		ConfigProvider: provider,
		// we're handling DisableGracefulShutdown via the cancelCtx being passed
		// to the collector's Run method in the Run function
		DisableGracefulShutdown: true,
	}, nil
}

func makeMapProvidersMap(providers ...confmap.Provider) map[string]confmap.Provider {
	ret := make(map[string]confmap.Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}
