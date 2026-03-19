package cmd

import (
	"fmt"
	"strconv"
	"time"
)

// parseTimestamp converts a user-supplied timestamp string to a Slack ts string
// (e.g. "1742378100.123456"). Accepts Slack ts format or RFC3339.
func parseTimestamp(s string) (string, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 1e9 {
		return s, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "", fmt.Errorf("cannot parse %q: use Slack ts (e.g. 1742378100.123456) or RFC3339 (e.g. 2024-01-15T10:00:00Z)", s)
	}
	return fmt.Sprintf("%d.000000", t.Unix()), nil
}
