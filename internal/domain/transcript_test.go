package domain_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestTranscript_WordCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{"empty", "", 0},
		{"one word", "hello", 1},
		{"multiple words", "hello world this is a test", 6},
		{"with newlines", "hello\nworld\ntest", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := domain.Transcript{Content: tt.content}
			assert.Equal(t, tt.want, tr.WordCount())
		})
	}
}

func TestTranscript_IsEmpty(t *testing.T) {
	assert.True(t, domain.Transcript{Content: ""}.IsEmpty())
	assert.True(t, domain.Transcript{Content: "   "}.IsEmpty())
	assert.True(t, domain.Transcript{Content: "\n\t"}.IsEmpty())
	assert.False(t, domain.Transcript{Content: "hello"}.IsEmpty())
}
