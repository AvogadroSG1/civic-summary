package domain_test

import (
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestMeeting_DateFolder(t *testing.T) {
	m := domain.Meeting{
		MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, "20250204", m.DateFolder())
}

func TestMeeting_ISODate(t *testing.T) {
	m := domain.Meeting{
		MeetingDate: time.Date(2025, 11, 25, 0, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, "2025-11-25", m.ISODate())
}

func TestMeeting_HumanDate(t *testing.T) {
	m := domain.Meeting{
		MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, "February 04, 2025", m.HumanDate())
}
