package cmd

import (
	"testing"
	"time"
)

func TestNormalizeTags(t *testing.T) {
	got := normalizeTags([]string{"API, backend", "api", "  ui "})
	want := []string{"api", "backend", "ui"}

	if len(got) != len(want) {
		t.Fatalf("tag count mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tag mismatch at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestParseDueDateInput(t *testing.T) {
	now := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)

	due, err := parseDueDateInput("today", now)
	if err != nil {
		t.Fatalf("parse today: %v", err)
	}
	if due.Hour() != 23 || due.Minute() != 59 {
		t.Fatalf("expected end-of-day for today, got %s", due.Format(time.RFC3339))
	}

	due, err = parseDueDateInput("+2d", now)
	if err != nil {
		t.Fatalf("parse +2d: %v", err)
	}
	if due.Day() != 20 {
		t.Fatalf("expected day 20 for +2d, got %s", due.Format("2006-01-02"))
	}

	due, err = parseDueDateInput("2026-03-01T14:30", now)
	if err != nil {
		t.Fatalf("parse absolute datetime: %v", err)
	}
	if due.Year() != 2026 || due.Month() != time.March || due.Day() != 1 || due.Hour() != 14 {
		t.Fatalf("unexpected due result: %s", due.Format(time.RFC3339))
	}
}

func TestParseDueFilterInput_DateBoundaries(t *testing.T) {
	now := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)

	afterCutoff, err := parseDueFilterInput("2026-03-01", now, false)
	if err != nil {
		t.Fatalf("parse due-after date: %v", err)
	}
	if afterCutoff.Hour() != 0 || afterCutoff.Minute() != 0 {
		t.Fatalf("expected start-of-day cutoff, got %s", afterCutoff.Format(time.RFC3339))
	}

	beforeCutoff, err := parseDueFilterInput("2026-03-01", now, true)
	if err != nil {
		t.Fatalf("parse due-before date: %v", err)
	}
	if beforeCutoff.Hour() != 23 || beforeCutoff.Minute() != 59 {
		t.Fatalf("expected end-of-day cutoff, got %s", beforeCutoff.Format(time.RFC3339))
	}
}
