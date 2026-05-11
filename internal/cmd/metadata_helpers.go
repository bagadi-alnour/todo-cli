package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func normalizeTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	tags := make([]string, 0, len(raw))
	for _, value := range raw {
		for _, token := range strings.Split(value, ",") {
			tag := strings.ToLower(strings.TrimSpace(token))
			if tag == "" {
				continue
			}
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			tags = append(tags, tag)
		}
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func mergeTags(existing []string, toAdd []string) []string {
	combined := append([]string{}, existing...)
	combined = append(combined, toAdd...)
	return normalizeTags(combined)
}

func removeTags(existing []string, toRemove []string) []string {
	if len(existing) == 0 || len(toRemove) == 0 {
		return normalizeTags(existing)
	}
	removals := make(map[string]struct{}, len(toRemove))
	for _, tag := range normalizeTags(toRemove) {
		removals[tag] = struct{}{}
	}
	if len(removals) == 0 {
		return normalizeTags(existing)
	}
	var out []string
	for _, tag := range normalizeTags(existing) {
		if _, remove := removals[tag]; !remove {
			out = append(out, tag)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseDueDateInput(input string, now time.Time) (*time.Time, error) {
	raw := strings.TrimSpace(strings.ToLower(input))
	if raw == "" {
		return nil, fmt.Errorf("due date cannot be empty")
	}

	switch raw {
	case "today":
		due := endOfDay(now)
		return &due, nil
	case "tomorrow":
		due := endOfDay(now.Add(24 * time.Hour))
		return &due, nil
	}

	if strings.HasPrefix(raw, "+") && len(raw) > 2 {
		amount, err := strconv.Atoi(raw[1 : len(raw)-1])
		if err == nil && amount >= 0 {
			unit := raw[len(raw)-1]
			switch unit {
			case 'h':
				due := now.Add(time.Duration(amount) * time.Hour)
				return &due, nil
			case 'd':
				due := endOfDay(now.Add(time.Duration(amount) * 24 * time.Hour))
				return &due, nil
			case 'w':
				due := endOfDay(now.Add(time.Duration(amount) * 7 * 24 * time.Hour))
				return &due, nil
			}
		}
	}

	if parsed, err := time.Parse(time.RFC3339, input); err == nil {
		due := parsed
		return &due, nil
	}

	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04"} {
		if parsed, err := time.ParseInLocation(layout, input, now.Location()); err == nil {
			due := parsed
			return &due, nil
		}
	}

	if parsed, err := time.ParseInLocation("2006-01-02", input, now.Location()); err == nil {
		due := endOfDay(parsed)
		return &due, nil
	}

	return nil, fmt.Errorf("invalid due date %q (use YYYY-MM-DD, YYYY-MM-DDTHH:MM, RFC3339, today, tomorrow, +2d, +1w, or +6h)", input)
}

func parseDueFilterInput(input string, now time.Time, endOfDayForDate bool) (time.Time, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return time.Time{}, fmt.Errorf("date filter cannot be empty")
	}
	if parsed, err := time.ParseInLocation("2006-01-02", raw, now.Location()); err == nil {
		if endOfDayForDate {
			return endOfDay(parsed), nil
		}
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	dueAt, err := parseDueDateInput(raw, now)
	if err != nil {
		return time.Time{}, err
	}
	return *dueAt, nil
}

func endOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, 0, t.Location())
}

func isOverdueDueDate(dueAt *time.Time, now time.Time) bool {
	if dueAt == nil {
		return false
	}
	return dueAt.Before(now)
}

func formatDueLabel(dueAt *time.Time, now time.Time) string {
	if dueAt == nil {
		return ""
	}
	if isOverdueDueDate(dueAt, now) {
		return fmt.Sprintf("OVERDUE (%s)", dueAt.Format("2006-01-02 15:04"))
	}
	return fmt.Sprintf("due %s", dueAt.Format("2006-01-02 15:04"))
}
