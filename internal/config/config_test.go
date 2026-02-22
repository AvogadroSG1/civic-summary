package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureConfig(t *testing.T) string {
	t.Helper()
	// Find fixtures relative to this test file.
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(wd, "..", "..", "testdata", "fixtures", "config.yaml")
}

func TestLoad_ValidConfig(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	assert.Equal(t, "/tmp/civic-summary-test", cfg.OutputDir)
	assert.Equal(t, 90, cfg.LogRetentionDays)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, []int{1, 2, 3}, cfg.BackoffDelays)
	assert.Equal(t, "yt-dlp", cfg.Tools.YtDlp)
	assert.Equal(t, "claude", cfg.Tools.Claude)
}

func TestLoad_Bodies(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	assert.Len(t, cfg.Bodies, 2)

	hagerstown, err := cfg.GetBody("hagerstown")
	require.NoError(t, err)
	assert.Equal(t, "hagerstown", hagerstown.Slug)
	assert.Equal(t, "Hagerstown City Council", hagerstown.Name)
	assert.Equal(t, "PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ", hagerstown.PlaylistID)
	assert.Contains(t, hagerstown.Tags, "City-Council")

	bocc, err := cfg.GetBody("bocc")
	require.NoError(t, err)
	assert.Equal(t, "bocc", bocc.Slug)
	assert.Equal(t, "Washington County Board of County Commissioners", bocc.Name)
}

func TestLoad_UnknownBody(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	_, err = cfg.GetBody("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown body")
}

func TestLoad_BodySlugs(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	slugs := cfg.BodySlugs()
	assert.Len(t, slugs, 2)
	assert.Contains(t, slugs, "hagerstown")
	assert.Contains(t, slugs, "bocc")
}

func TestLoad_Paths(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	body, _ := cfg.GetBody("hagerstown")
	assert.Equal(t,
		"/tmp/civic-summary-test/Hagerstown Town Council - Citizen Summary",
		cfg.BodyOutputDir(body),
	)
	assert.Equal(t,
		"/tmp/civic-summary-test/Hagerstown Town Council - Citizen Summary/Finalized Meeting Summaries",
		cfg.FinalizedDir(body),
	)
}

func TestValidate_MissingOutputDir(t *testing.T) {
	cfg := &config.Config{}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output_dir is required")
}

func TestValidate_NoBodies(t *testing.T) {
	cfg := &config.Config{OutputDir: "/tmp"}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one body")
}
