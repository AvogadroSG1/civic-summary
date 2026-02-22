package domain_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestBody_DiscoveryURL(t *testing.T) {
	t.Run("FromPlaylistID", func(t *testing.T) {
		b := domain.Body{PlaylistID: "PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ"}
		assert.Equal(t, "https://www.youtube.com/playlist?list=PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ", b.DiscoveryURL())
	})

	t.Run("FromVideoSourceURL", func(t *testing.T) {
		b := domain.Body{VideoSourceURL: "https://www.youtube.com/@washingtoncomd/streams"}
		assert.Equal(t, "https://www.youtube.com/@washingtoncomd/streams", b.DiscoveryURL())
	})

	t.Run("VideoSourceURLTakesPrecedence", func(t *testing.T) {
		b := domain.Body{
			PlaylistID:     "PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ",
			VideoSourceURL: "https://www.youtube.com/@washingtoncomd/streams",
		}
		assert.Equal(t, "https://www.youtube.com/@washingtoncomd/streams", b.DiscoveryURL())
	})

	t.Run("OnlyVideoSourceURL", func(t *testing.T) {
		b := domain.Body{VideoSourceURL: "https://www.youtube.com/@example/videos"}
		assert.Equal(t, "https://www.youtube.com/@example/videos", b.DiscoveryURL())
		assert.Empty(t, b.PlaylistID)
	})
}

func TestBody_VideoURL(t *testing.T) {
	b := domain.Body{}
	assert.Equal(t, "https://www.youtube.com/watch?v=abc123", b.VideoURL("abc123"))
}
