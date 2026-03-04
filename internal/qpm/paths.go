package qpm

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolvePackagesDir(quirinoJSONPath string, packagesDirName string) (string, error) {
	if packagesDirName == "" {
		packagesDirName = "QPackages"
	}

	if filepath.IsAbs(packagesDirName) {
		return "", fmt.Errorf("--packages-dir must be a directory name next to Quirino.json (got absolute path %q)", packagesDirName)
	}

	clean := filepath.Clean(packagesDirName)
	if clean == "." || clean == ".." {
		return "", fmt.Errorf("--packages-dir must be a directory name (got %q)", packagesDirName)
	}

	// Enforce "same level as Quirino.json": disallow any path separators.
	if filepath.Base(clean) != clean || strings.Contains(clean, "/") || strings.Contains(clean, "\\") {
		return "", fmt.Errorf("--packages-dir must not contain path separators (got %q)", packagesDirName)
	}

	workingDir := filepath.Dir(quirinoJSONPath)
	return filepath.Join(workingDir, clean), nil
}

