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

func TestCollectOutputs_Filtering(t *testing.T) {
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "example1.usp")
	splsWorkDir := filepath.Join(sourceDir, "SPlsWork")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("test"), 0o644)
	require.NoError(t, err)

	// Create SPlsWork directory
	err = os.MkdirAll(splsWorkDir, 0o755)
	require.NoError(t, err)

	// Create output files for multiple source files (simulating shared SPlsWork)
	splsWorkFiles := []string{
		// Files for example1.usp (should be collected)
		"example1.dll",
		"example1.cs",
		"example1.inf",
		"S2_example1.c",
		"S2_example1.h",
		"S2_example1.elf",
		"S2_example1.map",
		"S2_example1.o",
		"S2_example1.spl",
		// Files for example2.usp (should NOT be collected)
		"example2.dll",
		"example2.cs",
		"S2_example2.c",
		"S2_example2.h",
		// Shared library files (should NOT be collected)
		"Version.ini",
		"ManagedUtilities.dll",
		"SplusLibrary.dll",
	}

	for _, file := range splsWorkFiles {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte("content"), 0o644)
		require.NoError(t, err)
	}

	// Create .ush file adjacent to source
	ushFile := filepath.Join(sourceDir, "example1.ush")
	err = os.WriteFile(ushFile, []byte("header"), 0o644)
	require.NoError(t, err)

	// Collect outputs for example1.usp
	outputs, err := CollectOutputs(sourceFile)
	require.NoError(t, err)

	// Should collect: 1 .ush file + 9 SPlsWork files = 10 total
	expectedCount := 10
	assert.Len(t, outputs, expectedCount, "Should collect .ush + SPlsWork files for example1.usp")

	// Verify correct files are included
	outputMap := make(map[string]bool)
	for _, output := range outputs {
		outputMap[output] = true
	}

	// Check .ush file (no prefix)
	assert.True(t, outputMap["example1.ush"], "Should include example1.ush")

	// Check SPlsWork files (with SPlsWork/ prefix)
	assert.True(t, outputMap[filepath.Join("SPlsWork", "example1.dll")], "Should include SPlsWork/example1.dll")
	assert.True(t, outputMap[filepath.Join("SPlsWork", "example1.cs")], "Should include SPlsWork/example1.cs")
	assert.True(t, outputMap[filepath.Join("SPlsWork", "S2_example1.c")], "Should include SPlsWork/S2_example1.c")
	assert.True(t, outputMap[filepath.Join("SPlsWork", "S2_example1.h")], "Should include SPlsWork/S2_example1.h")

	// Verify incorrect files are excluded
	assert.False(t, outputMap[filepath.Join("SPlsWork", "example2.dll")], "Should NOT include example2.dll")
	assert.False(t, outputMap[filepath.Join("SPlsWork", "S2_example2.c")], "Should NOT include S2_example2.c")
	assert.False(t, outputMap[filepath.Join("SPlsWork", "Version.ini")], "Should NOT include shared library files")
	assert.False(t, outputMap[filepath.Join("SPlsWork", "ManagedUtilities.dll")], "Should NOT include shared library files")
}

func TestCache_StoreAndGet(t *testing.T) {
	// Create temp directories
	cacheDir := t.TempDir()
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "test.usp")
	splsWorkDir := filepath.Join(sourceDir, "SPlsWork")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("test source"), 0o644)
	require.NoError(t, err)

	// Create SPlsWork directory
	err = os.MkdirAll(splsWorkDir, 0o755)
	require.NoError(t, err)

	// Create fake output files matching the source file name in SPlsWork
	splsWorkOutputs := []string{"test.dll", "S2_test.elf", "S2_test.h"}
	for _, output := range splsWorkOutputs {
		path := filepath.Join(splsWorkDir, output)
		err := os.WriteFile(path, []byte("output content"), 0o644)
		require.NoError(t, err)
	}

	// Create .ush file adjacent to source
	ushFile := filepath.Join(sourceDir, "test.ush")
	err = os.WriteFile(ushFile, []byte("header content"), 0o644)
	require.NoError(t, err)

	// Create some unrelated files in SPlsWork (should be filtered out)
	unrelatedFiles := []string{"other.dll", "Version.ini"}
	for _, output := range unrelatedFiles {
		path := filepath.Join(splsWorkDir, output)
		err := os.WriteFile(path, []byte("unrelated content"), 0o644)
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
	err = cache.Store(sourceFile, cfg, true)
	require.NoError(t, err)

	// Cache hit now
	entry, err = cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	require.NotNil(t, entry, "Should be cache hit after storing")

	assert.Equal(t, sourceFile, entry.SourceFile)
	assert.Equal(t, "234", entry.Target)
	assert.True(t, entry.Success)
	assert.Len(t, entry.Outputs, 4, "Should cache 3 SPlsWork files + 1 .ush file")

	// Verify artifacts were copied (only the matching files)
	hash, _ := HashSource(sourceFile, cfg)
	artifactDir := filepath.Join(cacheDir, "artifacts", hash)

	// Check .ush file
	assert.FileExists(t, filepath.Join(artifactDir, "test.ush"), ".ush file should be cached")

	// Check SPlsWork files
	for _, output := range splsWorkOutputs {
		path := filepath.Join(artifactDir, "SPlsWork", output)
		assert.FileExists(t, path, "SPlsWork artifact should exist in cache")
	}

	// Verify unrelated files were NOT cached
	for _, output := range unrelatedFiles {
		path := filepath.Join(artifactDir, "SPlsWork", output)
		assert.NoFileExists(t, path, "Unrelated file should NOT be cached")
	}
}

func TestCache_Restore(t *testing.T) {
	// Create temp directories
	cacheDir := t.TempDir()
	sourceDir := t.TempDir()
	restoreDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "test.usp")
	splsWorkDir := filepath.Join(sourceDir, "SPlsWork")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("test source"), 0o644)
	require.NoError(t, err)

	// Create SPlsWork directory
	err = os.MkdirAll(splsWorkDir, 0o755)
	require.NoError(t, err)

	// Create fake output files in SPlsWork
	splsWorkOutputs := []string{"test.dll", "S2_test.elf"}
	for _, output := range splsWorkOutputs {
		path := filepath.Join(splsWorkDir, output)
		err := os.WriteFile(path, []byte("cached content"), 0o644)
		require.NoError(t, err)
	}

	// Create .ush file
	ushFile := filepath.Join(sourceDir, "test.ush")
	err = os.WriteFile(ushFile, []byte("header content"), 0o644)
	require.NoError(t, err)

	// Create cache and store
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	cfg := &config.Config{
		Target:      "234",
		UserFolders: []string{},
	}

	err = cache.Store(sourceFile, cfg, true)
	require.NoError(t, err)

	// Get entry
	entry, err := cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Restore to different directory
	err = cache.Restore(entry, restoreDir)
	require.NoError(t, err)

	// Verify .ush file was restored
	restoredUsh := filepath.Join(restoreDir, "test.ush")
	assert.FileExists(t, restoredUsh, ".ush file should be restored")
	content, err := os.ReadFile(restoredUsh)
	require.NoError(t, err)
	assert.Equal(t, "header content", string(content))

	// Verify SPlsWork files were restored
	for _, output := range splsWorkOutputs {
		path := filepath.Join(restoreDir, "SPlsWork", output)
		assert.FileExists(t, path, "SPlsWork file should be restored")

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "cached content", string(content))
	}
}

func TestCache_Clear(t *testing.T) {
	cacheDir := t.TempDir()
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "test.usp")
	splsWorkDir := filepath.Join(sourceDir, "SPlsWork")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("test source"), 0o644)
	require.NoError(t, err)

	// Create SPlsWork directory
	err = os.MkdirAll(splsWorkDir, 0o755)
	require.NoError(t, err)

	// Create output file in SPlsWork
	err = os.WriteFile(filepath.Join(splsWorkDir, "test.dll"), []byte("output"), 0o644)
	require.NoError(t, err)

	// Create cache and store
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	cfg := &config.Config{Target: "234"}

	err = cache.Store(sourceFile, cfg, true)
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
		sourceDir := t.TempDir()
		sourceFile := filepath.Join(sourceDir, "test.usp")
		splsWorkDir := filepath.Join(sourceDir, "SPlsWork")

		// Different content = different hash
		err := os.WriteFile(sourceFile, []byte(fmt.Sprintf("test %d", i)), 0o644)
		require.NoError(t, err)

		// Create SPlsWork directory (even if empty, so CollectOutputs doesn't fail)
		err = os.MkdirAll(splsWorkDir, 0o755)
		require.NoError(t, err)

		cfg := &config.Config{Target: "234"}
		err = cache.Store(sourceFile, cfg, true)
		require.NoError(t, err)
	}

	count, size, err = cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.GreaterOrEqual(t, size, int64(0))
}
