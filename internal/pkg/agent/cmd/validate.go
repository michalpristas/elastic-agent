// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/elastic/elastic-agent/internal/pkg/cli"
	"github.com/elastic/elastic-agent/internal/pkg/otel"
)

func newValidateCommandWithArgs(_ []string, _ *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "validate",
		Short:         "Validates the OpenTelemetry collector configuration without running the collector",
		SilenceUsage:  true, // do not display usage on error
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgFiles, err := getConfigFiles(cmd)
			if err != nil {
				return err
			}
			return validateOtelConfig(cmd.Context(), cfgFiles)
		},
	}

	return cmd
}

func validateOtelConfig(ctx context.Context, cfgFiles []string) error {
	return otel.Validate(ctx, cfgFiles)
}
