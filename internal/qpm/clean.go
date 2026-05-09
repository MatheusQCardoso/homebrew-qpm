package qpm

import (
	"context"
	"fmt"
	"os"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/fs"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
)

type CleanOptions struct {
	QuirinoJSONPath string
	PackagesDirName string
}

func Clean(ctx context.Context, opt CleanOptions) error {
	_ = ctx
	if opt.QuirinoJSONPath == "" {
		return fmt.Errorf("QuirinoJSONPath is required")
	}

	quirinoBytes, err := fs.ReadFile(opt.QuirinoJSONPath)
	if err != nil {
		return err
	}
	m, err := model.DecodeStrict[model.QuirinoManifest](quirinoBytes, "Quirino.json")
	if err != nil {
		return err
	}

	packagesDir, err := resolvePackagesDir(opt.QuirinoJSONPath, firstNonEmpty(opt.PackagesDirName, m.PackagesDir))
	if err != nil {
		return err
	}

	if _, err := os.Stat(packagesDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return fs.RemoveAll(packagesDir)
}
