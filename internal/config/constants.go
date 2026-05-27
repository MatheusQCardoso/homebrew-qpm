package config

import "time"

// Network & Timeout Configuration
const (
	// DefaultNetworkTimeout is the default timeout for network operations (in seconds).
	DefaultNetworkTimeout = 30 * time.Second
)

// Cache Configuration
const (
	// QPMCacheDirName is the name of the cache directory used internally.
	QPMCacheDirName = ".qpm-cache"

	// FetchTempDirPrefix is the prefix for temporary directories created during fetch operations.
	FetchTempDirPrefix = "fetch-"

	// ClonedRepoDirName is the name of the directory where repositories are cloned.
	ClonedRepoDirName = "repo"
)

// Manifest File Names
const (
	// SPMManifestFileName is the filename for Swift Package Manager manifests.
	SPMManifestFileName = "Package.swift"

	// QPMManifestFileExtension is the file extension for QPM package manifests.
	QPMManifestFileExtension = ".Package.json"
)

// Default Directory Names
const (
	// DefaultPackagesDirName is the default name for the packages directory.
	DefaultPackagesDirName = "QPackages"
)

// File System Permissions
const (
	// DirPermissions is the permission mode for created directories.
	DirPermissions = 0o755

	// FilePermissions is the permission mode for created files.
	FilePermissions = 0o644
)

// Directories to Skip in File Walks
var SkipDirs = []string{
	".git",
	".build",
	"DerivedData",
	"node_modules",
}
