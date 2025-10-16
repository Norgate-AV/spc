package config

import (
	"os"
	"path/filepath"
)

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
