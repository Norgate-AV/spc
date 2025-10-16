package utils

import (
	"strconv"
)

// ParseTarget parses target string into series slice
func ParseTarget(t string) []string {
	series := make([]string, 0)

	for _, r := range t {
		if s := int(r - '0'); s >= 2 && s <= 4 {
			series = append(series, "series"+strconv.Itoa(s))
		}
	}

	return series
}
