package qpm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/fs"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/git"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
)

type GitFetcher struct {
	cacheDir       string
	basePath       string
	git            git.Runner
	log            Logger
	networkTimeout time.Duration
}

func NewGitFetcher(packagesDir string, basePath string, log Logger, networkTimeout time.Duration) *GitFetcher {
	cacheDirectory := filepath.Join(packagesDir, ".qpm-cache")
	_ = fs.EnsureDir(cacheDirectory)
	if log == nil {
		log = NopLogger{}
	}
	return &GitFetcher{
		cacheDir:       cacheDirectory,
		basePath:       basePath,
		git:            git.Runner{Verbose: log.VerboseEnabled()},
		log:            log,
		networkTimeout: networkTimeout,
	}
}

func (f *GitFetcher) FetchQpmPackageJSON(ctx context.Context, moduleName string, spec model.DepSpec) ([]byte, error) {
	spec = spec.Normalized()
	if spec.Repo == "" {
		f.log.Infof("resolve %s (local qpm): %s", moduleName, spec.Path)

		resolvedPath := spec.Path
		if isRelativePathStr(spec.Path) {
			if f.basePath == "" {
				return nil, fmt.Errorf("relative path %q requires basePath to be set", spec.Path)
			}
			resolvedPath, err := resolveRelativePath(spec.Path, f.basePath)
			if err != nil {
				return nil, err
			}
			f.log.Verbosef("  resolved relative path: %s -> %s", spec.Path, resolvedPath)
		}

		localPackage, err := ResolveLocalQpmManifest(moduleName, resolvedPath)
		if err != nil {
			return nil, err
		}
		f.log.Verbosef("  manifest: %s", localPackage.ManifestPath)
		return fs.ReadFile(localPackage.ManifestPath)
	}
	rel := qpmPackageJSONRepoPath(moduleName, spec)
	f.log.Infof("resolve %s (qpm): %s [%s]", moduleName, spec.Repo, pickRef(spec))
	return f.fetchFile(ctx, spec, rel)
}

func (f *GitFetcher) FetchSpmPackageSwift(ctx context.Context, moduleName string, spec model.DepSpec) ([]byte, error) {
	spec = spec.Normalized()
	if spec.Repo == "" {
		f.log.Infof("resolve %s (local spm): %s", moduleName, spec.Path)

		resolvedPath := spec.Path
		if isRelativePathStr(spec.Path) {
			if f.basePath == "" {
				return nil, fmt.Errorf("relative path %q requires basePath to be set", spec.Path)
			}
			resolvedPath, err := resolveRelativePath(spec.Path, f.basePath)
			if err != nil {
				return nil, err
			}
			f.log.Verbosef("  resolved relative path: %s -> %s", spec.Path, resolvedPath)
		}

		localPackage, err := ResolveLocalSpmManifest(resolvedPath)
		if err != nil {
			return nil, err
		}
		f.log.Verbosef("  manifest: %s", localPackage.ManifestPath)
		return fs.ReadFile(localPackage.ManifestPath)
	}
	rel := spmPackageSwiftRepoPath(spec)
	f.log.Infof("resolve %s (spm): %s [%s]", moduleName, spec.Repo, pickRef(spec))
	return f.fetchFile(ctx, spec, rel)
}

func (f *GitFetcher) fetchFile(ctx context.Context, spec model.DepSpec, repoRelPath string) ([]byte, error) {
	spec = spec.Normalized()
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	temporaryRoot, err := os.MkdirTemp(f.cacheDir, "fetch-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(temporaryRoot) }()

	cloneDirectory := filepath.Join(temporaryRoot, "repo")

	ref := pickRef(spec)
	f.log.Verbosef("  sparse fetch: %s (ref=%s) paths=%v", spec.Repo, ref, []string{repoRelPath})
	fetchCtx := ctx
	var cancel context.CancelFunc
	if f.networkTimeout > 0 {
		fetchCtx, cancel = context.WithTimeout(ctx, f.networkTimeout)
		defer cancel()
	}
	if err := git.SparseClone(fetchCtx, f.git, cloneDirectory, git.SparseCloneOptions{
		Repo:        spec.Repo,
		Ref:         ref,
		SparsePaths: []string{repoRelPath},
	}); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, WrapErrorf(err, "fetch %s: timed out after %s", spec.Repo, f.networkTimeout)
		}
		return nil, err
	}

	localFilePath := filepath.Join(cloneDirectory, filepath.FromSlash(repoRelPath))
	f.log.Verbosef("  fetched file: %s", localFilePath)
	return fs.ReadFile(localFilePath)
}

func qpmPackageJSONRepoPath(moduleName string, spec model.DepSpec) string {
	if spec.Path != "" {
		return path.Join(spec.Path, moduleName+".Package.json")
	}
	return moduleName + ".Package.json"
}

func spmPackageSwiftRepoPath(spec model.DepSpec) string {
	if spec.Path != "" {
		return path.Join(spec.Path, "Package.swift")
	}
	return "Package.swift"
}

func pickRef(spec model.DepSpec) string {
	switch {
	case spec.Branch != "":
		return spec.Branch
	case spec.Tag != "":
		return spec.Tag
	case spec.Revision != "":
		return spec.Revision
	default:
		return ""
	}
}

func isRelativePathStr(pathString string) bool {
	if pathString == "" {
		return false
	}
	if strings.HasPrefix(pathString, "/") || strings.HasPrefix(pathString, "~") {
		return false
	}
	return strings.HasPrefix(pathString, ".") || strings.HasPrefix(pathString, "./") || strings.Contains(pathString, "/..")
}

func (f *GitFetcher) String() string {
	return fmt.Sprintf("GitFetcher(cacheDir=%s)", f.cacheDir)
}
