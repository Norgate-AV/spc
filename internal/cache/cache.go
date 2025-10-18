// Package cache provides build caching functionality for SIMPL+ compilation.
//
// The cache system addresses the challenge of shared output directories (SPlsWork)
// where the Crestron compiler places artifacts from multiple source files together.
// Rather than caching entire directories, the cache:
//
//  1. Filters artifacts by source file name (e.g., example1.dll, S2_example1.c)
//  2. Stores only relevant artifacts per source file in separate cache entries
//  3. Uses SHA256 hashing of source content + configuration for cache keys
//  4. Stores metadata in BoltDB and artifacts in the filesystem
//
// This allows incremental compilation where each source file can be cached
// and restored independently, even when multiple files share the same output directory.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"

	"github.com/Norgate-AV/spc/internal/config"
)

const (
	// DefaultCacheDir is the default cache directory name
	DefaultCacheDir = ".spc-cache"

	// bucketName is the BoltDB bucket name for cache entries
	bucketName = "builds"
)

// Cache manages build artifacts and metadata using BoltDB
type Cache struct {
	db   *bbolt.DB
	root string // Root directory for cache (.spc-cache/)
}

// New creates a new cache instance
// If cacheDir is empty, uses DefaultCacheDir in current working directory
func New(cacheDir string) (*Cache, error) {
	if cacheDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}

		cacheDir = filepath.Join(cwd, DefaultCacheDir)
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Open BoltDB
	dbPath := filepath.Join(cacheDir, "cache.db")
	db, err := bbolt.Open(dbPath, 0o600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	// Create bucket if it doesn't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create cache bucket: %w", err)
	}

	return &Cache{
		db:   db,
		root: cacheDir,
	}, nil
}

// Close closes the cache database
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}

	return nil
}

// Get retrieves a cache entry by source file and configuration
// Returns nil if cache miss
func (c *Cache) Get(sourceFile string, cfg *config.Config) (*Entry, error) {
	hash, err := HashSource(sourceFile, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to hash source: %w", err)
	}

	var entry Entry
	err = c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		data := b.Get([]byte(hash))
		if data == nil {
			return nil // Cache miss
		}

		return json.Unmarshal(data, &entry)
	})
	if err != nil {
		return nil, err
	}

	if entry.Hash == "" {
		return nil, nil // Cache miss
	}

	return &entry, nil
}

// Store saves a cache entry and copies artifacts
func (c *Cache) Store(sourceFile string, cfg *config.Config, success bool) error {
	hash, err := HashSource(sourceFile, cfg)
	if err != nil {
		return fmt.Errorf("failed to hash source: %w", err)
	}

	// Collect outputs from both source dir and SPlsWork dir
	// Only collect files for the current target (prevents caching leftover files)
	outputs, err := CollectOutputs(sourceFile, cfg.Target)
	if err != nil {
		return fmt.Errorf("failed to collect outputs: %w", err)
	}

	// Create cache entry
	entry := Entry{
		Hash:            hash,
		SourceFile:      sourceFile,
		Target:          cfg.Target,
		CompilerVersion: "", // TODO: detect compiler version
		UserFolders:     cfg.UserFolders,
		Timestamp:       time.Now(),
		Outputs:         outputs,
		Success:         success,
	}

	// Store metadata in BoltDB
	err = c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		return b.Put([]byte(hash), data)
	})
	if err != nil {
		return fmt.Errorf("failed to store cache entry: %w", err)
	}

	// Copy artifacts to cache (outputs are relative to source directory)
	if success && len(outputs) > 0 {
		artifactDir := c.artifactDir(hash)
		sourceDir := filepath.Dir(sourceFile)
		if err := CopyArtifacts(sourceDir, artifactDir, outputs); err != nil {
			return fmt.Errorf("failed to copy artifacts: %w", err)
		}
	}

	// Cache shared files (only once, if not already cached)
	if success {
		sourceDir := filepath.Dir(sourceFile)
		if err := c.cacheSharedFiles(sourceDir); err != nil {
			// Don't fail the whole operation if shared files caching fails
			fmt.Fprintf(os.Stderr, "Warning: Failed to cache shared files: %v\n", err)
		}
	}

	return nil
}

// cacheSharedFiles caches shared library files if not already cached
func (c *Cache) cacheSharedFiles(sourceDir string) error {
	sharedDir := filepath.Join(c.root, "shared")

	// Collect shared files that need to be cached
	sharedFiles, err := CollectSharedFiles(sourceDir)
	if err != nil || len(sharedFiles) == 0 {
		return err
	}

	// Check which shared files are missing from cache
	var missingFiles []string
	for _, file := range sharedFiles {
		cachedFile := filepath.Join(sharedDir, file)
		if _, err := os.Stat(cachedFile); os.IsNotExist(err) {
			missingFiles = append(missingFiles, file)
		}
	}

	// If all files already cached, skip
	if len(missingFiles) == 0 {
		return nil
	}

	// Copy missing shared files to cache
	if err := CopyArtifacts(sourceDir, sharedDir, missingFiles); err != nil {
		return fmt.Errorf("failed to copy shared files: %w", err)
	}

	return nil
}

// Restore copies cached artifacts back to the source directory
func (c *Cache) Restore(entry *Entry, destDir string) error {
	if !entry.Success || len(entry.Outputs) == 0 {
		return fmt.Errorf("cannot restore failed build or build with no outputs")
	}

	// Restore source-specific artifacts
	artifactDir := c.artifactDir(entry.Hash)
	if err := RestoreArtifacts(artifactDir, destDir, entry.Outputs); err != nil {
		return err
	}

	// Restore shared files if needed (if SPlsWork exists but shared files are missing)
	if err := c.restoreSharedFiles(destDir); err != nil {
		// Don't fail if shared files restoration fails - they might already exist
		// or will be recreated on next full compile
		fmt.Fprintf(os.Stderr, "Warning: Failed to restore shared files: %v\n", err)
	}

	return nil
}

// restoreSharedFiles restores shared library files if they're missing
func (c *Cache) restoreSharedFiles(destDir string) error {
	sharedDir := filepath.Join(c.root, "shared")

	// Check if we have cached shared files
	if _, err := os.Stat(sharedDir); os.IsNotExist(err) {
		return nil // No shared files cached, skip
	}

	// Check if shared files already exist in destination
	splsWorkDir := filepath.Join(destDir, "SPlsWork")
	if needsSharedFiles, err := checkSharedFilesExist(splsWorkDir); err != nil || !needsSharedFiles {
		return err // Either error or files already exist
	}

	// Collect what shared files we have cached
	entries, err := os.ReadDir(filepath.Join(sharedDir, "SPlsWork"))
	if err != nil {
		return err
	}

	var sharedFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			sharedFiles = append(sharedFiles, filepath.Join("SPlsWork", entry.Name()))
		}
	}

	// Restore shared files
	return RestoreArtifacts(sharedDir, destDir, sharedFiles)
}

// checkSharedFilesExist checks if shared files are missing from SPlsWork
// Returns true if shared files need to be restored
func checkSharedFilesExist(splsWorkDir string) (bool, error) {
	// If SPlsWork doesn't exist, we definitely need shared files
	if _, err := os.Stat(splsWorkDir); os.IsNotExist(err) {
		return true, nil
	}

	// Check for presence of at least one common shared file
	commonSharedFiles := []string{"Version.ini", "ManagedUtilities.dll", "SplusLibrary.dll"}
	for _, file := range commonSharedFiles {
		if _, err := os.Stat(filepath.Join(splsWorkDir, file)); err == nil {
			return false, nil // At least one shared file exists, assume others are there
		}
	}

	// No shared files found, need to restore them
	return true, nil
}

// Clear removes all cache entries and artifacts
func (c *Cache) Clear() error {
	// Clear BoltDB
	err := c.db.Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket([]byte(bucketName))
	})
	if err != nil {
		return err
	}

	// Recreate bucket
	err = c.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucket([]byte(bucketName))
		return err
	})
	if err != nil {
		return err
	}

	// Remove artifacts directory
	artifactsDir := filepath.Join(c.root, "artifacts")
	if err := os.RemoveAll(artifactsDir); err != nil {
		return fmt.Errorf("failed to remove artifacts: %w", err)
	}

	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats() (int, int64, error) {
	var count int
	var totalSize int64

	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		count = b.Stats().KeyN
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	// Calculate total artifact size
	artifactsDir := filepath.Join(c.root, "artifacts")
	err = filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	return count, totalSize, nil
}

// artifactDir returns the directory path for a given cache hash
func (c *Cache) artifactDir(hash string) string {
	return filepath.Join(c.root, "artifacts", hash)
}
