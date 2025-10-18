package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	// Test 1: Collect outputs for target "234" (all series)
	outputs, err := CollectOutputs(sourceFile, "234")
	require.NoError(t, err)

	// Should collect: 1 .ush file + 9 SPlsWork files = 10 total
	expectedCount := 10
	assert.Len(t, outputs, expectedCount, "Should collect .ush + SPlsWork files for example1.usp with target 234")

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

	// Test 2: Collect outputs for target "34" (no series 2)
	outputs34, err := CollectOutputs(sourceFile, "34")
	require.NoError(t, err)

	// Should collect: 1 .ush file + 3 SPlsWork files (no S2_* files) = 4 total
	expectedCount34 := 4
	assert.Len(t, outputs34, expectedCount34, "Should collect only Series 3/4 files for target 34")

	outputMap34 := make(map[string]bool)
	for _, output := range outputs34 {
		outputMap34[output] = true
	}

	// Check that Series 3/4 files are included
	assert.True(t, outputMap34["example1.ush"], "Should include example1.ush")
	assert.True(t, outputMap34[filepath.Join("SPlsWork", "example1.dll")], "Should include example1.dll")
	assert.True(t, outputMap34[filepath.Join("SPlsWork", "example1.cs")], "Should include example1.cs")
	assert.True(t, outputMap34[filepath.Join("SPlsWork", "example1.inf")], "Should include example1.inf")

	// Check that Series 2 files are NOT included
	assert.False(t, outputMap34[filepath.Join("SPlsWork", "S2_example1.c")], "Should NOT include S2_example1.c for target 34")
	assert.False(t, outputMap34[filepath.Join("SPlsWork", "S2_example1.h")], "Should NOT include S2_example1.h for target 34")
	assert.False(t, outputMap34[filepath.Join("SPlsWork", "S2_example1.elf")], "Should NOT include S2_example1.elf for target 34")
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

// TestCache_DifferentTargets verifies that different targets create different cache entries
func TestCache_DifferentTargets(t *testing.T) {
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

	// Create cache
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	// Store same source file with different targets
	targets := []string{"2", "3", "4", "23", "34", "234"}
	hashes := make(map[string]string)

	for _, target := range targets {
		cfg := &config.Config{
			Target:      target,
			UserFolders: []string{},
		}

		// Create target-specific output file
		outputFile := filepath.Join(splsWorkDir, fmt.Sprintf("test_%s.dll", target))
		err := os.WriteFile(outputFile, []byte(fmt.Sprintf("output for %s", target)), 0o644)
		require.NoError(t, err)

		// Store in cache
		err = cache.Store(sourceFile, cfg, true)
		require.NoError(t, err)

		// Get hash for this target
		hash, err := HashSource(sourceFile, cfg)
		require.NoError(t, err)
		hashes[target] = hash

		// Verify we can retrieve the entry
		entry, err := cache.Get(sourceFile, cfg)
		require.NoError(t, err)
		require.NotNil(t, entry, "Should find cache entry for target %s", target)
		assert.Equal(t, target, entry.Target)
	}

	// Verify all targets produced different hashes
	uniqueHashes := make(map[string]bool)
	for _, hash := range hashes {
		uniqueHashes[hash] = true
	}
	assert.Equal(t, len(targets), len(uniqueHashes), "Each target should produce a unique hash")

	// Verify cache stats show all entries
	count, _, err := cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, len(targets), count, "Should have one entry per target")
}

// TestCache_SharedFiles_IncrementalCaching verifies that shared files are cached incrementally
// when building with different targets (e.g., series2 first, then series3)
func TestCache_SharedFiles_IncrementalCaching(t *testing.T) {
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

	// Create cache
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	// Simulate series2 build (creates only Version.ini as shared file)
	series2SharedFiles := []string{"Version.ini"}
	for _, file := range series2SharedFiles {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte("series2 shared"), 0o644)
		require.NoError(t, err)
	}

	// Create series2 source-specific files
	series2Files := []string{"S2_test.c", "S2_test.h", "S2_test.elf"}
	for _, file := range series2Files {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte("series2 output"), 0o644)
		require.NoError(t, err)
	}

	cfg2 := &config.Config{Target: "2", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg2, true)
	require.NoError(t, err)

	// Verify Version.ini was cached as shared file
	sharedDir := filepath.Join(cacheDir, "shared", "SPlsWork")
	assert.FileExists(t, filepath.Join(sharedDir, "Version.ini"), "Version.ini should be cached")

	// Count shared files after series2 (should be 1)
	entries, err := os.ReadDir(sharedDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "Should have only Version.ini after series2 build")

	// Simulate series3 build (creates .NET DLLs + config files as shared files)
	series3SharedFiles := []string{
		"ManagedUtilities.dll",
		"SimplSharpHelperInterface.dll",
		"SplusLibrary.dll",
		"Simpl#Config.xml",
		"SimplSharpData.dat",
	}
	for _, file := range series3SharedFiles {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte("series3 shared"), 0o644)
		require.NoError(t, err)
	}

	// Create series3 source-specific files
	series3Files := []string{"test.cs", "test.dll", "test.inf"}
	for _, file := range series3Files {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte("series3 output"), 0o644)
		require.NoError(t, err)
	}

	cfg3 := &config.Config{Target: "3", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg3, true)
	require.NoError(t, err)

	// Verify all shared files are now cached (Version.ini + 5 series3 files = 6 total)
	entries, err = os.ReadDir(sharedDir)
	require.NoError(t, err)
	assert.Len(t, entries, 6, "Should have Version.ini + 5 series3 shared files")

	// Verify specific files exist
	for _, file := range series3SharedFiles {
		assert.FileExists(t, filepath.Join(sharedDir, file), "%s should be cached", file)
	}
	assert.FileExists(t, filepath.Join(sharedDir, "Version.ini"), "Version.ini should still be cached")
}

// TestCache_SharedFiles_Restoration verifies that shared files are restored correctly
func TestCache_SharedFiles_Restoration(t *testing.T) {
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

	// Create shared library files
	sharedFiles := []string{
		"Version.ini",
		"ManagedUtilities.dll",
		"SimplSharpHelperInterface.dll",
	}
	for _, file := range sharedFiles {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte(fmt.Sprintf("content of %s", file)), 0o644)
		require.NoError(t, err)
	}

	// Create source-specific files
	sourceSpecificFiles := []string{"test.cs", "test.dll", "test.inf"}
	for _, file := range sourceSpecificFiles {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte("source-specific content"), 0o644)
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

	cfg := &config.Config{Target: "3", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg, true)
	require.NoError(t, err)

	// Get entry
	entry, err := cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Restore to different directory
	err = cache.Restore(entry, restoreDir)
	require.NoError(t, err)

	// Verify source-specific files were restored
	for _, file := range sourceSpecificFiles {
		path := filepath.Join(restoreDir, "SPlsWork", file)
		assert.FileExists(t, path, "%s should be restored", file)
	}

	// Verify .ush file was restored
	assert.FileExists(t, filepath.Join(restoreDir, "test.ush"), ".ush file should be restored")

	// Verify shared library files were restored
	for _, file := range sharedFiles {
		path := filepath.Join(restoreDir, "SPlsWork", file)
		assert.FileExists(t, path, "Shared file %s should be restored", file)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("content of %s", file), string(content))
	}
}

// TestCache_MixedTargets_Isolation verifies that mixed target builds (e.g., "23")
// maintain proper cache isolation from single-target builds
func TestCache_MixedTargets_Isolation(t *testing.T) {
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

	// Create cache
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	// Build for series2 only
	series2File := filepath.Join(splsWorkDir, "S2_test.elf")
	err = os.WriteFile(series2File, []byte("series2 output"), 0o644)
	require.NoError(t, err)

	cfg2 := &config.Config{Target: "2", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg2, true)
	require.NoError(t, err)

	// Build for series3 only
	series3File := filepath.Join(splsWorkDir, "test.dll")
	err = os.WriteFile(series3File, []byte("series3 output"), 0o644)
	require.NoError(t, err)

	cfg3 := &config.Config{Target: "3", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg3, true)
	require.NoError(t, err)

	// Build for series2+3 mixed
	series23Both := []string{
		"S2_test.elf",
		"test.dll",
	}
	for _, file := range series23Both {
		path := filepath.Join(splsWorkDir, file)
		err := os.WriteFile(path, []byte("mixed output"), 0o644)
		require.NoError(t, err)
	}

	cfg23 := &config.Config{Target: "23", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg23, true)
	require.NoError(t, err)

	// Verify all three builds created separate cache entries
	hash2, _ := HashSource(sourceFile, cfg2)
	hash3, _ := HashSource(sourceFile, cfg3)
	hash23, _ := HashSource(sourceFile, cfg23)

	assert.NotEqual(t, hash2, hash3, "Series2 and Series3 should have different hashes")
	assert.NotEqual(t, hash2, hash23, "Series2 and Series23 should have different hashes")
	assert.NotEqual(t, hash3, hash23, "Series3 and Series23 should have different hashes")

	// Verify we can retrieve each entry independently
	entry2, err := cache.Get(sourceFile, cfg2)
	require.NoError(t, err)
	require.NotNil(t, entry2)
	assert.Equal(t, "2", entry2.Target)

	entry3, err := cache.Get(sourceFile, cfg3)
	require.NoError(t, err)
	require.NotNil(t, entry3)
	assert.Equal(t, "3", entry3.Target)

	entry23, err := cache.Get(sourceFile, cfg23)
	require.NoError(t, err)
	require.NotNil(t, entry23)
	assert.Equal(t, "23", entry23.Target)
}

// TestCache_SharedFiles_NotDuplicated verifies that shared files are not duplicated
// when the same shared file is encountered in multiple builds
func TestCache_SharedFiles_NotDuplicated(t *testing.T) {
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

	// Create cache
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	// Create shared file with specific content
	sharedFile := filepath.Join(splsWorkDir, "Version.ini")
	originalContent := "original version content"
	err = os.WriteFile(sharedFile, []byte(originalContent), 0o644)
	require.NoError(t, err)

	// Create source-specific file
	sourceSpecific := filepath.Join(splsWorkDir, "test.dll")
	err = os.WriteFile(sourceSpecific, []byte("output"), 0o644)
	require.NoError(t, err)

	// Store first build
	cfg1 := &config.Config{Target: "3", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg1, true)
	require.NoError(t, err)

	// Verify shared file was cached
	cachedSharedFile := filepath.Join(cacheDir, "shared", "SPlsWork", "Version.ini")
	assert.FileExists(t, cachedSharedFile)
	content, err := os.ReadFile(cachedSharedFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(content))

	// Get file info for later comparison
	info1, err := os.Stat(cachedSharedFile)
	require.NoError(t, err)

	// Modify the shared file (simulating a second build that might have different content)
	modifiedContent := "modified version content"
	err = os.WriteFile(sharedFile, []byte(modifiedContent), 0o644)
	require.NoError(t, err)

	// Store second build with different target (should NOT overwrite cached shared file)
	cfg2 := &config.Config{Target: "4", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg2, true)
	require.NoError(t, err)

	// Verify cached shared file was NOT overwritten (should still have original content)
	content, err = os.ReadFile(cachedSharedFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(content), "Cached shared file should not be overwritten")

	// Verify file timestamp didn't change (file wasn't re-written)
	info2, err := os.Stat(cachedSharedFile)
	require.NoError(t, err)
	assert.Equal(t, info1.ModTime(), info2.ModTime(), "Shared file should not be re-cached")
}

// TestCache_UshFiles_TargetSpecific verifies that .ush files are cached per-target
// and the correct .ush file is restored for each target
func TestCache_UshFiles_TargetSpecific(t *testing.T) {
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

	// Create cache
	cache, err := New(cacheDir)
	require.NoError(t, err)
	defer cache.Close()

	// Test with different targets and different .ush content
	type targetTest struct {
		target     string
		ushContent string
		dllFile    string
	}

	tests := []targetTest{
		{target: "2", ushContent: "USH content for series 2 with SMWRev=2.02.05", dllFile: "S2_test.elf"},
		{target: "3", ushContent: "USH content for series 3 with SMWRev=3.00.00", dllFile: "test.dll"},
		{target: "34", ushContent: "USH content for series 34 combined", dllFile: "test.dll"},
	}

	cachedUshContents := make(map[string]string)

	// Cache builds for each target with different .ush files
	for _, tt := range tests {
		// Create target-specific .ush file
		ushFile := filepath.Join(sourceDir, "test.ush")
		err := os.WriteFile(ushFile, []byte(tt.ushContent), 0o644)
		require.NoError(t, err)

		// Create target-specific output file
		outputFile := filepath.Join(splsWorkDir, tt.dllFile)
		err = os.WriteFile(outputFile, []byte(fmt.Sprintf("output for %s", tt.target)), 0o644)
		require.NoError(t, err)

		cfg := &config.Config{Target: tt.target, UserFolders: []string{}}

		// Store in cache
		err = cache.Store(sourceFile, cfg, true)
		require.NoError(t, err)

		// Remember what we cached
		cachedUshContents[tt.target] = tt.ushContent

		// Verify it was cached
		hash, err := HashSource(sourceFile, cfg)
		require.NoError(t, err)
		cachedUshPath := filepath.Join(cacheDir, "artifacts", hash, "test.ush")
		assert.FileExists(t, cachedUshPath, ".ush should be cached for target %s", tt.target)

		content, err := os.ReadFile(cachedUshPath)
		require.NoError(t, err)
		assert.Equal(t, tt.ushContent, string(content), "Cached .ush content should match for target %s", tt.target)

		// Clean up for next iteration
		err = os.Remove(ushFile)
		require.NoError(t, err)
		err = os.Remove(outputFile)
		require.NoError(t, err)
	}

	// Now verify restoration - each target should restore its own .ush file
	for _, tt := range tests {
		cfg := &config.Config{Target: tt.target, UserFolders: []string{}}

		// Get cache entry
		entry, err := cache.Get(sourceFile, cfg)
		require.NoError(t, err)
		require.NotNil(t, entry, "Should have cache entry for target %s", tt.target)

		// Restore to a clean directory
		restoreDir := t.TempDir()
		err = cache.Restore(entry, restoreDir)
		require.NoError(t, err)

		// Verify the correct .ush file was restored
		restoredUshPath := filepath.Join(restoreDir, "test.ush")
		assert.FileExists(t, restoredUshPath, ".ush should be restored for target %s", tt.target)

		content, err := os.ReadFile(restoredUshPath)
		require.NoError(t, err)
		assert.Equal(t, cachedUshContents[tt.target], string(content),
			"Restored .ush should match cached content for target %s", tt.target)

		// Verify it's the target-specific content, not from another target
		for otherTarget, otherContent := range cachedUshContents {
			if otherTarget != tt.target {
				assert.NotEqual(t, otherContent, string(content),
					"Target %s .ush should not contain target %s content", tt.target, otherTarget)
			}
		}
	}
}

// TestCache_Restore_SkipsIdenticalFiles verifies that restoration only copies files
// when necessary, skipping identical files to improve performance
func TestCache_Restore_SkipsIdenticalFiles(t *testing.T) {
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

	// Create output files
	outputs := []string{"test.dll", "test.cs"}
	for _, output := range outputs {
		path := filepath.Join(splsWorkDir, output)
		err := os.WriteFile(path, []byte(fmt.Sprintf("content of %s", output)), 0o644)
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

	cfg := &config.Config{Target: "3", UserFolders: []string{}}
	err = cache.Store(sourceFile, cfg, true)
	require.NoError(t, err)

	// Get entry
	entry, err := cache.Get(sourceFile, cfg)
	require.NoError(t, err)
	require.NotNil(t, entry)

	// First restoration (files don't exist) - should copy all files
	restoreDir1 := t.TempDir()
	err = cache.Restore(entry, restoreDir1)
	require.NoError(t, err)

	// Verify files were created
	for _, output := range outputs {
		assert.FileExists(t, filepath.Join(restoreDir1, "SPlsWork", output))
	}
	assert.FileExists(t, filepath.Join(restoreDir1, "test.ush"))

	// Get timestamps of restored files
	dllPath := filepath.Join(restoreDir1, "SPlsWork", "test.dll")
	infoBeforeSecondRestore, err := os.Stat(dllPath)
	require.NoError(t, err)

	// Wait a moment to ensure timestamps would differ if file was rewritten
	time.Sleep(10 * time.Millisecond)

	// Second restoration (files already exist and are identical) - should skip copying
	err = cache.Restore(entry, restoreDir1)
	require.NoError(t, err)

	// Verify file timestamp didn't change (file wasn't copied)
	infoAfterSecondRestore, err := os.Stat(dllPath)
	require.NoError(t, err)
	assert.Equal(t, infoBeforeSecondRestore.ModTime(), infoAfterSecondRestore.ModTime(),
		"File should not be copied when it's already identical")

	// Now modify a file to make it different
	err = os.WriteFile(dllPath, []byte("corrupted content"), 0o644)
	require.NoError(t, err)

	infoAfterModification, err := os.Stat(dllPath)
	require.NoError(t, err)

	// Third restoration (file exists but differs) - should copy the modified file
	time.Sleep(10 * time.Millisecond)
	err = cache.Restore(entry, restoreDir1)
	require.NoError(t, err)

	// Verify file was restored (timestamp changed and content correct)
	infoAfterThirdRestore, err := os.Stat(dllPath)
	require.NoError(t, err)
	assert.NotEqual(t, infoAfterModification.ModTime(), infoAfterThirdRestore.ModTime(),
		"File should be copied when it differs from cached version")

	// Verify content was correctly restored
	content, err := os.ReadFile(dllPath)
	require.NoError(t, err)
	assert.Equal(t, "content of test.dll", string(content), "Content should be restored correctly")
}
