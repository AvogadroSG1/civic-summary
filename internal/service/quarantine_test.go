package service_test

import (
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quarantineTestConfig(t *testing.T) (*config.Config, domain.Body) {
	t.Helper()
	cfg := &config.Config{
		OutputDir: t.TempDir(),
		Bodies: map[string]domain.Body{
			"test": {
				Slug:         "test",
				OutputSubdir: "Test",
			},
		},
	}
	body, _ := cfg.GetBody("test")
	return cfg, body
}

func TestQuarantineService_RoundTrip(t *testing.T) {
	cfg, body := quarantineTestConfig(t)
	svc := service.NewQuarantineService(cfg)

	meeting := domain.Meeting{
		VideoID:     "abc123",
		MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
		BodySlug:    "test",
	}

	// Quarantine.
	err := svc.Quarantine(body, meeting, "test error", "", "partial output")
	require.NoError(t, err)

	// List.
	entries, err := svc.ListQuarantined(body)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "abc123", entries[0].VideoID)
	assert.Equal(t, "2025-02-04", entries[0].MeetingDate)
	assert.Equal(t, "test error", entries[0].Error)
	assert.Equal(t, 0, entries[0].RetryCount)

	// Increment retry.
	err = svc.IncrementRetry(body, "abc123")
	require.NoError(t, err)

	entries, err = svc.ListQuarantined(body)
	require.NoError(t, err)
	assert.Equal(t, 1, entries[0].RetryCount)

	// Remove.
	err = svc.Remove(body, "abc123")
	require.NoError(t, err)

	entries, err = svc.ListQuarantined(body)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestQuarantineService_ListEmpty(t *testing.T) {
	cfg, body := quarantineTestConfig(t)
	svc := service.NewQuarantineService(cfg)

	entries, err := svc.ListQuarantined(body)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestQuarantineService_MultipleEntries(t *testing.T) {
	cfg, body := quarantineTestConfig(t)
	svc := service.NewQuarantineService(cfg)

	for _, id := range []string{"vid1", "vid2", "vid3"} {
		meeting := domain.Meeting{
			VideoID:     id,
			MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
			BodySlug:    "test",
		}
		require.NoError(t, svc.Quarantine(body, meeting, "error", "", ""))
	}

	entries, err := svc.ListQuarantined(body)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}
