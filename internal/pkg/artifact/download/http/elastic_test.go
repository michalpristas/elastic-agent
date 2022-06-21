// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"context"
	"crypto/sha512"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent/internal/pkg/agent/program"
	"github.com/elastic/elastic-agent/internal/pkg/artifact"
	"github.com/elastic/elastic-agent/pkg/core/logger"
)

const (
	version       = "7.5.1"
	sourcePattern = "/downloads/beats/filebeat/"
	source        = "http://artifacts.elastic.co/downloads/"
)

var (
	beatSpec = program.Spec{
		Name:     "filebeat",
		Cmd:      "filebeat",
		Artifact: "beats/filebeat",
	}
)

type testCase struct {
	system string
	arch   string
}

func TestDownload(t *testing.T) {
	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	log, _ := logger.New("", false)
	timeout := 30 * time.Second
	testCases := getTestCases()
	elasticClient := getElasticCoClient()

	config := &artifact.Config{
		SourceURI:       source,
		TargetDirectory: targetDir,
		HTTPTransportSettings: httpcommon.HTTPTransportSettings{
			Timeout: timeout,
		},
	}

	for _, testCase := range testCases {
		testName := fmt.Sprintf("%s-binary-%s", testCase.system, testCase.arch)
		t.Run(testName, func(t *testing.T) {
			config.OperatingSystem = testCase.system
			config.Architecture = testCase.arch

			testClient := NewDownloaderWithClient(log, config, elasticClient)
			artifactPath, err := testClient.Download(context.Background(), beatSpec, version)
			if err != nil {
				t.Fatal(err)
			}

			_, err = os.Stat(artifactPath)
			if err != nil {
				t.Fatal(err)
			}

			os.Remove(artifactPath)
		})
	}
}

func TestVerify(t *testing.T) {
	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	log, _ := logger.New("", false)
	timeout := 30 * time.Second
	testCases := getRandomTestCases()
	elasticClient := getElasticCoClient()

	config := &artifact.Config{
		SourceURI:       source,
		TargetDirectory: targetDir,
		HTTPTransportSettings: httpcommon.HTTPTransportSettings{
			Timeout: timeout,
		},
	}

	for _, testCase := range testCases {
		testName := fmt.Sprintf("%s-binary-%s", testCase.system, testCase.arch)
		t.Run(testName, func(t *testing.T) {
			config.OperatingSystem = testCase.system
			config.Architecture = testCase.arch

			testClient := NewDownloaderWithClient(log, config, elasticClient)
			artifact, err := testClient.Download(context.Background(), beatSpec, version)
			if err != nil {
				t.Fatal(err)
			}

			_, err = os.Stat(artifact)
			if err != nil {
				t.Fatal(err)
			}

			testVerifier, err := NewVerifier(config, true, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = testVerifier.Verify(beatSpec, version)
			require.NoError(t, err)

			os.Remove(artifact)
			os.Remove(artifact + ".sha512")
		})
	}
}

func getTestCases() []testCase {
	// always test random package to save time
	return []testCase{
		{"linux", "32"},
		{"linux", "64"},
		{"linux", "arm64"},
		{"darwin", "32"},
		{"darwin", "64"},
		{"windows", "32"},
		{"windows", "64"},
	}
}

//nolint:gosec,G404 // this is just for unit tests secure random number is not needed
func getRandomTestCases() []testCase {
	tt := getTestCases()

	rand.Seed(time.Now().UnixNano())
	first := rand.Intn(len(tt))
	second := rand.Intn(len(tt))

	return []testCase{
		tt[first],
		tt[second],
	}
}

func getElasticCoClient() http.Client {
	correctValues := map[string]struct{}{
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "i386.deb"):             {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "amd64.deb"):            {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "i686.rpm"):             {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "x86_64.rpm"):           {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "linux-x86.tar.gz"):     {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "linux-arm64.tar.gz"):   {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "linux-x86_64.tar.gz"):  {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "windows-x86.zip"):      {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "windows-x86_64.zip"):   {},
		fmt.Sprintf("%s-%s-%s", beatSpec.CommandName(), version, "darwin-x86_64.tar.gz"): {},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		packageName := r.URL.Path[len(sourcePattern):]
		isShaReq := strings.HasSuffix(packageName, ".sha512")
		packageName = strings.TrimSuffix(packageName, ".sha512")

		if _, ok := correctValues[packageName]; !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}

		content := []byte(packageName)
		if isShaReq {
			hash := sha512.Sum512(content)
			_, err := w.Write([]byte(fmt.Sprintf("%x %s", hash, packageName)))
			if err != nil {
				panic(err)
			}
		} else {
			_, err := w.Write(content)
			if err != nil {
				panic(err)
			}
		}
	})
	server := httptest.NewServer(handler)

	return http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, server.Listener.Addr().String())
			},
		},
	}
}
