package cache

import "time"

// Entry represents a cached build result
type Entry struct {
	// Hash is the unique identifier for this cache entry
	// Computed from: source file content + target + compiler version + user folders
	Hash string `json:"hash"`

	// SourceFile is the absolute path to the source .usp file
	SourceFile string `json:"source_file"`

	// Target is the compilation target (e.g., "234")
	Target string `json:"target"`

	// CompilerVersion is the version of SPlusCC.exe used
	CompilerVersion string `json:"compiler_version"`

	// UserFolders are the include paths used during compilation
	UserFolders []string `json:"user_folders"`

	// Timestamp when this entry was created
	Timestamp time.Time `json:"timestamp"`

	// Outputs lists the compiled artifact files (relative to SPlsWork/)
	Outputs []string `json:"outputs"`

	// Success indicates if the build was successful
	Success bool `json:"success"`
}
