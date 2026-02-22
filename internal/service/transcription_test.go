package service_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestTranscriptionService_ValidateTranscript(t *testing.T) {
	mock := executor.NewMockCommander()
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	svc := service.NewTranscriptionService(ytdlp, nil)

	tests := []struct {
		name       string
		transcript domain.Transcript
		wantErr    bool
	}{
		{
			"empty transcript",
			domain.Transcript{Content: ""},
			true,
		},
		{
			"too short",
			domain.Transcript{Content: "just a few words here"},
			true,
		},
		{
			"valid transcript",
			domain.Transcript{Content: generateWords(600)},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateTranscript(tt.transcript)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func generateWords(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			result += " "
		}
		result += "word"
	}
	return result
}
