package qpm

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type LocalPackage struct {
	RootDir      string
	ManifestPath string
}

func ResolveLocalQpmManifest(moduleName string, basePath string) (LocalPackage, error) {
	basePath, err := expandTilde(basePath)
	if err != nil {
		return LocalPackage{}, err
	}
	filename := moduleName + ".Package.json"
	p, err := findFile(basePath, filename)
	if err != nil {
		return LocalPackage{}, err
	}
	return LocalPackage{
		RootDir:      filepath.Dir(p),
		ManifestPath: p,
	}, nil
}

func ResolveLocalSpmManifest(basePath string) (LocalPackage, error) {
	basePath, err := expandTilde(basePath)
	if err != nil {
		return LocalPackage{}, err
	}
	p, err := findFile(basePath, "Package.swift")
	if err != nil {
		return LocalPackage{}, err
	}
	return LocalPackage{
		RootDir:      filepath.Dir(p),
		ManifestPath: p,
	}, nil
}

func expandTilde(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", nil
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			return home, nil
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

var errFound = errors.New("found")

func findFile(basePath string, filename string) (string, error) {
	if basePath == "" {
		return "", fmt.Errorf("path is required for local dependency")
	}

	info, err := os.Stat(basePath)
	if err == nil && !info.IsDir() {
		if filepath.Base(basePath) == filename {
			return basePath, nil
		}
		return "", fmt.Errorf("expected %s but got file %s", filename, basePath)
	}

	// First: direct child.
	direct := filepath.Join(basePath, filename)
	if _, err := os.Stat(direct); err == nil {
		return direct, nil
	}

	var found string
	walkErr := filepath.WalkDir(basePath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".build", "DerivedData", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == filename {
			found = p
			return errFound
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errFound) {
		return "", walkErr
	}
	if found == "" {
		return "", fmt.Errorf("could not find %s under %s", filename, basePath)
	}
	return found, nil
}

