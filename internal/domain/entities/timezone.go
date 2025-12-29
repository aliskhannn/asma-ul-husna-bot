package entities

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTimezoneLocation supports:
// - IANA tz like "Europe/Moscow"
// - "UTC" / "GMT"
// - fixed offsets: "UTC+3", "UTC-7", "UTC+5:30", "+3", "-03:30"
//
// For fixed offsets it returns time.FixedZone, which is DST-agnostic. [web:342]
func ParseTimezoneLocation(tz string) (*time.Location, error) {
	tz = strings.TrimSpace(tz)
	if tz == "" || strings.EqualFold(tz, "UTC") || strings.EqualFold(tz, "Etc/UTC") {
		return time.UTC, nil
	}
	if strings.EqualFold(tz, "GMT") {
		return time.UTC, nil
	}

	// 1) Try IANA first
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc, nil
	}

	// 2) Try UTC offset formats
	offSec, ok := parseUTCOffsetSeconds(tz)
	if !ok {
		return nil, fmt.Errorf("unsupported timezone %q", tz)
	}
	name := formatUTCOffsetName(offSec)      // e.g. "UTC+03:00"
	return time.FixedZone(name, offSec), nil // fixed offset location [web:342][web:350]
}

func parseUTCOffsetSeconds(tz string) (int, bool) {
	s := strings.TrimSpace(tz)

	// Allow "+3", "-03:30"
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		return parseSignHourMinuteToSeconds(s)
	}

	// Allow "UTC+3", "UTC-7", "UTC+5:30"
	if strings.HasPrefix(strings.ToUpper(s), "UTC") {
		s = strings.TrimSpace(s[3:])
		if s == "" {
			return 0, true
		}
		if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
			return parseSignHourMinuteToSeconds(s)
		}
		return 0, false
	}

	return 0, false
}

func parseSignHourMinuteToSeconds(s string) (int, bool) {
	if len(s) < 2 {
		return 0, false
	}
	sign := 1
	if s[0] == '+' {
		sign = 1
	} else if s[0] == '-' {
		sign = -1
	} else {
		return 0, false
	}
	s = s[1:]

	hh := s
	mm := "0"
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, false
		}
		hh, mm = parts[0], parts[1]
	}

	h, err := strconv.Atoi(hh)
	if err != nil {
		return 0, false
	}
	m, err := strconv.Atoi(mm)
	if err != nil {
		return 0, false
	}

	// basic sanity
	if h < 0 || h > 14 || m < 0 || m >= 60 {
		return 0, false
	}

	return sign * (h*3600 + m*60), true
}

func formatUTCOffsetName(offsetSec int) string {
	sign := "+"
	if offsetSec < 0 {
		sign = "-"
		offsetSec = -offsetSec
	}
	h := offsetSec / 3600
	m := (offsetSec % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, h, m)
}
