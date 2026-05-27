package qpm

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/config"
)

func resolvePackagesDir(quirinoJSONPath string, packagesDirName string) (string, error) {
	if packagesDirName == "" {
		packagesDirName = config.DefaultPackagesDirName
	}

	if filepath.IsAbs(packagesDirName) {
		return "", fmt.Errorf("packagesDir must be a directory name next to Quirino.json (got absolute path %q)", packagesDirName)
	}

	clean := filepath.Clean(packagesDirName)
	if clean == "." || clean == ".." {
		return "", fmt.Errorf("packagesDir must be a directory name (got %q)", packagesDirName)
	}

	if filepath.Base(clean) != clean || strings.Contains(clean, "/") || strings.Contains(clean, "\\") {
		return "", fmt.Errorf("packagesDir must not contain path separators (got %q)", packagesDirName)
	}

	workingDir := filepath.Dir(quirinoJSONPath)
	return filepath.Join(workingDir, clean), nil
}
