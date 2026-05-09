package qpm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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

	packagesDir, err := resolvePackagesDir(opt.QuirinoJSONPath, firstNonEmpty(opt.PackagesDirName, m.PackagesDir))
	if err != nil {
		return nil, err
	}
	if err := fs.EnsureDir(packagesDir); err != nil {
		return nil, err
	}

	basePath := filepath.Dir(opt.QuirinoJSONPath)
	fetcher := NewGitFetcher(packagesDir, basePath, NopLogger{})
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

	packagesDir, err := resolvePackagesDir(opt.QuirinoJSONPath, firstNonEmpty(opt.PackagesDirName, m.PackagesDir))
	if err != nil {
		return err
	}
	log.Infof("packages dir: %s", packagesDir)
	if err := fs.EnsureDir(packagesDir); err != nil {
		return err
	}

	log.Infof("building dependency graph...")
	basePath := filepath.Dir(opt.QuirinoJSONPath)
	fetcher := NewGitFetcher(packagesDir, basePath, log)
	g, err := graph.Build(ctx, m, fetcher)
	if err != nil {
		return err
	}
	log.Infof("resolved %d packages", len(g.Nodes))

	lockPath := filepath.Join(packagesDir, "qpm.lock.json")
	lockJSON, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	if err := fs.WriteFile(lockPath, lockJSON); err != nil {
		return err
	}
	log.Verbosef("wrote lockfile: %s", lockPath)

	start := time.Now()
	log.Infof("installing...")
	installer := NewInstaller(packagesDir, log)
	if err := installer.InstallAll(ctx, g); err != nil {
		return err
	}
	duration := time.Since(start).Round(time.Second)
	log.Infof("done")
	printInstallSummary(log, g, duration, packagesDir)
	return nil
}

func printInstallSummary(log Logger, g *graph.Graph, duration time.Duration, packagesDir string) {
	total, qpmCount, spmCount := installSummaryCounts(g)
	size, err := directorySize(packagesDir)
	if err != nil {
		log.Verbosef("unable to compute packages directory size: %v", err)
	}
	log.Infof("\nInstalled %d packages", total)
	log.Infof("  • QPM: %d", qpmCount)
	log.Infof("  • SPM: %d", spmCount)
	if err == nil {
		log.Infof("  • Total size: %s", formatBytes(size))
	}
	log.Infof("  • Time: %s", formatDuration(duration))
}

func directorySize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func installSummaryCounts(g *graph.Graph) (total, qpmCount, spmCount int) {
	if g == nil {
		return 0, 0, 0
	}
	total = len(g.Nodes)
	for _, spec := range g.Nodes {
		switch spec.Type {
		case model.DependencyTypeQPM:
			qpmCount++
		case model.DependencyTypeSPM:
			spmCount++
		}
	}
	return total, qpmCount, spmCount
}

func formatDuration(d time.Duration) string {
	return d.String()
}

func formatBytes(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
