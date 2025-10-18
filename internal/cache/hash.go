package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Norgate-AV/spc/internal/config"
)

// HashSource creates a unique hash for a source file and its build configuration
// The hash is based on:
// - Source file content
// - Target series
// - Compiler version (TODO: detect from SPlusCC.exe)
// - User folders (sorted for consistency)
func HashSource(sourceFile string, cfg *config.Config) (string, error) {
	h := sha256.New()

	// Hash source file content
	f, err := os.Open(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash source file: %w", err)
	}

	// Hash target
	h.Write([]byte(cfg.Target))

	// Hash user folders (sorted for consistency)
	sortedFolders := make([]string, len(cfg.UserFolders))
	copy(sortedFolders, cfg.UserFolders)
	sort.Strings(sortedFolders)
	h.Write([]byte(strings.Join(sortedFolders, "|")))

	// TODO: Hash compiler version
	// For now, we assume compiler version doesn't change
	// In future, detect version from SPlusCC.exe

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashFile creates a hash of a file's content
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
