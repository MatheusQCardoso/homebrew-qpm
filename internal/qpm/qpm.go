package qpm

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/fs"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/graph"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
)

type InstallOptions struct {
	QuirinoJSONPath string
	PackagesDirName string
	Verbose         bool
}

type BuildGraphOptions struct {
	QuirinoJSONPath string
	PackagesDirName string
}

func BuildGraphJSON(ctx context.Context, opt BuildGraphOptions) ([]byte, error) {
	if opt.QuirinoJSONPath == "" {
		return nil, fmt.Errorf("QuirinoJSONPath is required")
	}

	quirinoBytes, err := fs.ReadFile(opt.QuirinoJSONPath)
	if err != nil {
		return nil, err
	}
	m, err := model.DecodeStrict[model.QuirinoManifest](quirinoBytes, "Quirino.json")
	if err != nil {
		return nil, err
	}

	packagesDir, err := resolvePackagesDir(opt.QuirinoJSONPath, opt.PackagesDirName)
	if err != nil {
		return nil, err
	}
	if err := fs.EnsureDir(packagesDir); err != nil {
		return nil, err
	}

	fetcher := NewGitFetcher(packagesDir, NopLogger{})
	g, err := graph.Build(ctx, m, fetcher)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(g, "", "  ")
}

func Install(ctx context.Context, opt InstallOptions) error {
	if opt.QuirinoJSONPath == "" {
		return fmt.Errorf("QuirinoJSONPath is required")
	}

	log := NewStdLogger(opt.Verbose)
	log.Infof("qpm install")
	log.Verbosef("  Quirino.json: %s", opt.QuirinoJSONPath)

	quirinoBytes, err := fs.ReadFile(opt.QuirinoJSONPath)
	if err != nil {
		return err
	}
	m, err := model.DecodeStrict[model.QuirinoManifest](quirinoBytes, "Quirino.json")
	if err != nil {
		return err
	}

	packagesDir, err := resolvePackagesDir(opt.QuirinoJSONPath, opt.PackagesDirName)
	if err != nil {
		return err
	}
	log.Infof("packages dir: %s", packagesDir)
	if err := fs.EnsureDir(packagesDir); err != nil {
		return err
	}

	log.Infof("building dependency graph...")
	fetcher := NewGitFetcher(packagesDir, log)
	g, err := graph.Build(ctx, m, fetcher)
	if err != nil {
		return err
	}
	log.Infof("resolved %d packages", len(g.Nodes))

	// Persist the graph for debugging/reproducibility.
	lockPath := filepath.Join(packagesDir, "qpm.lock.json")
	lockJSON, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	if err := fs.WriteFile(lockPath, lockJSON); err != nil {
		return err
	}
	log.Verbosef("wrote lockfile: %s", lockPath)

	log.Infof("installing...")
	installer := NewInstaller(packagesDir, log)
	if err := installer.InstallAll(ctx, g); err != nil {
		return err
	}
	log.Infof("done")
	return nil
}

