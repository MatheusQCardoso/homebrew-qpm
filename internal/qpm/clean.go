package qpm

import (
	"context"
	"fmt"
	"os"

	"github.com/matheusqcardoso/qpm/internal/fs"
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

	packagesDir, err := resolvePackagesDir(opt.QuirinoJSONPath, opt.PackagesDirName)
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

