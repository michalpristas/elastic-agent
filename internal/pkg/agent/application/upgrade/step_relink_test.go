package upgrade

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/elastic-agent/internal/pkg/agent/application/paths"
	"github.com/stretchr/testify/require"
)

func TestChangeSymlink(t *testing.T) {
	tempTop := t.TempDir()
	defer os.RemoveAll(tempTop)
	targetHash := "123456"
	oldHash := "654321"

	// create symlink to elastic-agent-{hash}
	hashedDir := fmt.Sprintf("%s-%s", agentName, targetHash)
	oldHashedDir := fmt.Sprintf("%s-%s", agentName, oldHash)

	symlinkPath := filepath.Join(tempTop, agentName)

	// paths.BinaryPath properly derives the binary directory depending on the platform. The path to the binary for macOS is inside of the app bundle.
	newPath := paths.BinaryPath(filepath.Join(tempTop, "data", hashedDir), agentName)
	oldPath := paths.BinaryPath(filepath.Join(tempTop, "data", oldHashedDir), agentName)

	prevNewPath := prevSymlinkPath(tempTop)

	// prepare
	os.MkdirAll(filepath.Dir(newPath), 0775)
	ioutil.WriteFile(newPath, []byte("new"), 0664)
	os.MkdirAll(filepath.Dir(oldPath), 0775)
	ioutil.WriteFile(oldPath, []byte("old"), 0664)
	os.Symlink(oldPath, symlinkPath)

	o, err := filepath.EvalSymlinks(symlinkPath)
	require.NoError(t, err)
	require.Equal(t, oldPath, o)

	err = changeSymlink(context.Background(), newErrorLogger(t), symlinkPath, newPath, prevNewPath)
	require.NoError(t, err)

	n, err := filepath.EvalSymlinks(symlinkPath)
	require.NoError(t, err)
	require.Equal(t, newPath, n)
}
