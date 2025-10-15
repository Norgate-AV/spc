package utils

import (
	"os"
	"path/filepath"
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

// FindLocalConfig finds local config file by walking up directories
func FindLocalConfig(dir string) string {
	for {
		for _, ext := range []string{"yml", "yaml", "json", "toml"} {
			path := filepath.Join(dir, ".spc."+ext)

			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return ""
}
