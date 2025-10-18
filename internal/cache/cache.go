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
func (c *Cache) Store(sourceFile string, cfg *config.Config, outputDir string, success bool) error {
	hash, err := HashSource(sourceFile, cfg)
	if err != nil {
		return fmt.Errorf("failed to hash source: %w", err)
	}

	// Collect outputs from build directory
	outputs, err := CollectOutputs(outputDir)
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

	// Copy artifacts to cache
	if success && len(outputs) > 0 {
		artifactDir := c.artifactDir(hash)
		if err := CopyArtifacts(outputDir, artifactDir, outputs); err != nil {
			return fmt.Errorf("failed to copy artifacts: %w", err)
		}
	}

	return nil
}

// Restore copies cached artifacts back to the output directory
func (c *Cache) Restore(entry *Entry, destDir string) error {
	if !entry.Success || len(entry.Outputs) == 0 {
		return fmt.Errorf("cannot restore failed build or build with no outputs")
	}

	artifactDir := c.artifactDir(entry.Hash)
	return RestoreArtifacts(artifactDir, destDir, entry.Outputs)
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
