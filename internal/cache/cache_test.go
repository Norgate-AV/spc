package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Norgate-AV/spc/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashSource(t *testing.T) {
	// Create temp file
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "test.usp")
	err := os.WriteFile(sourceFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	cfg := &config.Config{
		Target:      "234",
		UserFolders: []string{"/path/to/folder1", "/path/to/folder2"},
	}

	// Hash should be consistent
	hash1, err := HashSource(sourceFile, cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	hash2, err := HashSource(sourceFile, cfg)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2, "Hash should be consistent")

	// Different content = different hash
	err = os.WriteFile(sourceFile, []byte("different content"), 0o644)
	require.NoError(t, err)

	hash3, err := HashSource(sourceFile, cfg)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3, "Different content should produce different hash")

	// Different target = different hash
	cfg2 := &config.Config{
		Target:      "3",
		UserFolders: []string{"/path/to/folder1", "/path/to/folder2"},
	}

	hash4, err := HashSource(sourceFile, cfg2)
	require.NoError(t, err)
	assert.NotEqual(t, hash3, hash4, "Different target should produce different hash")

	// User folders order shouldn't matter (sorted internally)
	cfg3 := &config.Config{
		Target:      "234",
		UserFolders: []string{"/path/to/folder2", "/path/to/folder1"}, // Reversed
	}

	// Reset file to original content
	err = os.WriteFile(sourceFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	hash5, err := HashSource(sourceFile, cfg3)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash5, "User folders should be sorted, order shouldn't matter")
}

func TestCache_StoreAndGet(t *testing.T) {
	// Create temp directories
	cacheDir := t.TempDir()
	outputDir := t.TempDir()
	sourceFile := filepath.Join(t.TempDir(), "test.usp")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("test source"), 0o644)
	require.NoError(t, err)

	// Create fake output files
	outputs := []string{"test.dll", "test.elf", "test.h"}
	for _, output := range outputs {
		path := filepath.Join(outputDir, output)
		err := os.WriteFile(path, []byte("output content"), 0o644)
		require.NoError(t, err)
	}

	// Create cache
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	cfg := &config.Config{
		Target:      "234",
		UserFolders: []string{},
	}

	// Cache miss initially
	entry, err := cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	assert.Nil(t, entry, "Should be cache miss initially")

	// Store in cache
	err = cache.Store(sourceFile, cfg, outputDir, true)
	require.NoError(t, err)

	// Cache hit now
	entry, err = cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	require.NotNil(t, entry, "Should be cache hit after storing")

	assert.Equal(t, sourceFile, entry.SourceFile)
	assert.Equal(t, "234", entry.Target)
	assert.True(t, entry.Success)
	assert.Len(t, entry.Outputs, 3)

	// Verify artifacts were copied
	hash, _ := HashSource(sourceFile, cfg)
	artifactDir := filepath.Join(cacheDir, "artifacts", hash)
	for _, output := range outputs {
		path := filepath.Join(artifactDir, output)
		assert.FileExists(t, path, "Artifact should exist in cache")
	}
}

func TestCache_Restore(t *testing.T) {
	// Create temp directories
	cacheDir := t.TempDir()
	outputDir := t.TempDir()
	restoreDir := t.TempDir()
	sourceFile := filepath.Join(t.TempDir(), "test.usp")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("test source"), 0o644)
	require.NoError(t, err)

	// Create fake output files
	outputs := []string{"test.dll", "test.elf"}
	for _, output := range outputs {
		path := filepath.Join(outputDir, output)
		err := os.WriteFile(path, []byte("cached content"), 0o644)
		require.NoError(t, err)
	}

	// Create cache and store
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	cfg := &config.Config{
		Target:      "234",
		UserFolders: []string{},
	}

	err = cache.Store(sourceFile, cfg, outputDir, true)
	require.NoError(t, err)

	// Get entry
	entry, err := cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Restore to different directory
	err = cache.Restore(entry, restoreDir)
	require.NoError(t, err)

	// Verify files were restored
	for _, output := range outputs {
		path := filepath.Join(restoreDir, output)
		assert.FileExists(t, path, "File should be restored")

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "cached content", string(content))
	}
}

func TestCache_Clear(t *testing.T) {
	cacheDir := t.TempDir()
	sourceFile := filepath.Join(t.TempDir(), "test.usp")
	outputDir := t.TempDir()

	// Create source file
	err := os.WriteFile(sourceFile, []byte("test source"), 0o644)
	require.NoError(t, err)

	// Create output file
	err = os.WriteFile(filepath.Join(outputDir, "test.dll"), []byte("output"), 0o644)
	require.NoError(t, err)

	// Create cache and store
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	cfg := &config.Config{Target: "234"}

	err = cache.Store(sourceFile, cfg, outputDir, true)
	require.NoError(t, err)

	// Verify entry exists
	entry, err := cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	assert.NotNil(t, entry)

	// Clear cache
	err = cache.Clear()
	require.NoError(t, err)

	// Verify entry is gone
	entry, err = cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	assert.Nil(t, entry, "Cache should be empty after clear")

	// Verify artifacts directory is gone
	artifactsDir := filepath.Join(cacheDir, "artifacts")
	_, err = os.Stat(artifactsDir)
	assert.True(t, os.IsNotExist(err), "Artifacts directory should be removed")
}

func TestCache_Stats(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	// Initially empty
	count, size, err := cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, int64(0), size)

	// Add some entries with different content (so different hashes)
	for i := 0; i < 3; i++ {
		sourceFile := filepath.Join(t.TempDir(), "test.usp")
		outputDir := t.TempDir()

		// Different content = different hash
		err := os.WriteFile(sourceFile, []byte(fmt.Sprintf("test %d", i)), 0o644)
		require.NoError(t, err)

		cfg := &config.Config{Target: "234"}
		err = cache.Store(sourceFile, cfg, outputDir, true)
		require.NoError(t, err)
	}

	count, size, err = cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.GreaterOrEqual(t, size, int64(0))
}
