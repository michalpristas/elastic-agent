// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	control "github.com/elastic/elastic-agent/internal/pkg/agent/control"
	"github.com/elastic/elastic-agent/internal/pkg/agent/control/v2/client"
	"github.com/elastic/elastic-agent/internal/pkg/agent/errors"
	"github.com/elastic/elastic-agent/internal/pkg/cli"
)

const (
	flagSourceURI    = "source-uri"
	flagSkipVerify   = "skip-verify"
	flagPGPBytes     = "pgp-bytes"
	flagPGPBytesPath = "pgp-bytes-path"
)

func newUpgradeCommandWithArgs(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade <version>",
		Short: "Upgrade the currently running Elastic Agent to the specified version",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			if err := upgradeCmd(streams, c, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringP(flagSourceURI, "s", "", "Source URI to download the new version from")
	cmd.Flags().BoolP(flagSkipVerify, "", false, "Skips package verification")
	cmd.Flags().String(flagPGPBytes, "", "PGP to use for package verification")
	cmd.Flags().String(flagPGPBytesPath, "", "Path to a file containing PGP to use for package verification")

	return cmd
}

func upgradeCmd(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	version := args[0]
	sourceURI, _ := cmd.Flags().GetString(flagSourceURI)

	c := client.New()
	err := c.Connect(context.Background())
	if err != nil {
		return errors.New(err, "Failed communicating to running daemon", errors.TypeNetwork, errors.M("socket", control.Address()))
	}
	defer c.Disconnect()

	skipVerification, _ := cmd.Flags().GetBool(flagSkipVerify)
	var pgpChecks []string
	if !skipVerification {
		pgpPath, _ := cmd.Flags().GetString(flagPGPBytesPath)
		if len(pgpPath) > 0 {
			content, err := ioutil.ReadFile(pgpPath)
			if err != nil {
				return errors.New(err, "failed to read pgp file")
			}
			if len(content) > 0 {
				pgpChecks = append(pgpChecks, string(content))
			}
		}

		pgpBytes, _ := cmd.Flags().GetString(flagPGPBytes)
		if len(pgpBytes) > 0 {
			pgpChecks = append(pgpChecks, pgpBytes)
		}
	}

	version, err = c.Upgrade(context.Background(), version, sourceURI, skipVerification, pgpChecks...)
	if err != nil {
		return errors.New(err, "Failed trigger upgrade of daemon")
	}
	fmt.Fprintf(streams.Out, "Upgrade triggered to version %s, Elastic Agent is currently restarting\n", version)
	return nil
}
