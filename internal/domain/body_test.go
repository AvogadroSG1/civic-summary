package domain_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestBody_PlaylistURL(t *testing.T) {
	b := domain.Body{PlaylistID: "PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ"}
	assert.Equal(t, "https://www.youtube.com/playlist?list=PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ", b.PlaylistURL())
}

func TestBody_VideoURL(t *testing.T) {
	b := domain.Body{}
	assert.Equal(t, "https://www.youtube.com/watch?v=abc123", b.VideoURL("abc123"))
}
