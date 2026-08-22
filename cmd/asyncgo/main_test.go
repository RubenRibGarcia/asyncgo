package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVersion(t *testing.T) {
	t.Run("should_return_injected_version", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "v1.2.3"
		assert.Equal(t, "v1.2.3", resolveVersion())
	})

	t.Run("should_fall_back_to_devel_when_untagged", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "devel"
		// A `go test` binary carries "(devel)" as its main module version.
		assert.Equal(t, "devel", resolveVersion())
	})
}

func TestRunVersion(t *testing.T) {
	old := version
	defer func() { version = old }()
	version = "v1.2.3"

	out := captureStdout(t, func() {
		require.NoError(t, run([]string{"version"}))
	})
	assert.Equal(t, "asyncgo v1.2.3\n", out)
}

func TestRunVersionWithCommit(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	defer func() { version, commit, date = oldVersion, oldCommit, oldDate }()
	version, commit, date = "v1.2.3", "abc1234", "2026-08-18T00:00:00Z"

	out := captureStdout(t, func() {
		require.NoError(t, run([]string{"version"}))
	})
	assert.Equal(t, "asyncgo v1.2.3 (commit abc1234, built 2026-08-18T00:00:00Z)\n", out)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}
