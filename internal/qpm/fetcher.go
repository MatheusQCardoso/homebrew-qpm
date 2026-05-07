package qpm

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/fs"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/git"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
)

type GitFetcher struct {
	cacheDir string
	basePath string // Directory of Quirino.json for resolving relative paths
	git      git.Runner
	log      Logger
}

func NewGitFetcher(packagesDir string, basePath string, log Logger) *GitFetcher {
	cacheDir := filepath.Join(packagesDir, ".qpm-cache")
	_ = fs.EnsureDir(cacheDir)
	if log == nil {
		log = NopLogger{}
	}
	return &GitFetcher{
		cacheDir: cacheDir,
		basePath: basePath,
		git:      git.Runner{Verbose: log.VerboseEnabled()},
		log:      log,
	}
}

func (f *GitFetcher) FetchQpmPackageJSON(ctx context.Context, moduleName string, spec model.DepSpec) ([]byte, error) {
	spec = spec.Normalized()
	if spec.Repo == "" {
		f.log.Infof("resolve %s (local qpm): %s", moduleName, spec.Path)
		
		// Handle relative paths by resolving them relative to basePath
		resolvePath := spec.Path
		if isRelativePathStr(spec.Path) {
			if f.basePath == "" {
				return nil, fmt.Errorf("relative path %q requires basePath to be set", spec.Path)
			}
			resolved, err := resolveRelativePath(spec.Path, f.basePath)
			if err != nil {
				return nil, err
			}
			resolvePath = resolved
			f.log.Verbosef("  resolved relative path: %s -> %s", spec.Path, resolved)
		}
		
		lp, err := ResolveLocalQpmManifest(moduleName, resolvePath)
		if err != nil {
			return nil, err
		}
		f.log.Verbosef("  manifest: %s", lp.ManifestPath)
		return fs.ReadFile(lp.ManifestPath)
	}
	rel := qpmPackageJSONRepoPath(moduleName, spec)
	f.log.Infof("resolve %s (qpm): %s [%s]", moduleName, spec.Repo, pickRef(spec))
	return f.fetchFile(ctx, spec, rel)
}

func (f *GitFetcher) FetchSpmPackageSwift(ctx context.Context, moduleName string, spec model.DepSpec) ([]byte, error) {
	spec = spec.Normalized()
	if spec.Repo == "" {
		f.log.Infof("resolve %s (local spm): %s", moduleName, spec.Path)
		
		// Handle relative paths by resolving them relative to basePath
		resolvePath := spec.Path
		if isRelativePathStr(spec.Path) {
			if f.basePath == "" {
				return nil, fmt.Errorf("relative path %q requires basePath to be set", spec.Path)
			}
			resolved, err := resolveRelativePath(spec.Path, f.basePath)
			if err != nil {
				return nil, err
			}
			resolvePath = resolved
			f.log.Verbosef("  resolved relative path: %s -> %s", spec.Path, resolved)
		}
		
		lp, err := ResolveLocalSpmManifest(resolvePath)
		if err != nil {
			return nil, err
		}
		f.log.Verbosef("  manifest: %s", lp.ManifestPath)
		return fs.ReadFile(lp.ManifestPath)
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

	tmpRoot, err := os.MkdirTemp(f.cacheDir, "fetch-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	cloneDir := filepath.Join(tmpRoot, "repo")

	ref := pickRef(spec)
	f.log.Verbosef("  sparse fetch: %s (ref=%s) paths=%v", spec.Repo, ref, []string{repoRelPath})
	if err := git.SparseClone(ctx, f.git, cloneDir, git.SparseCloneOptions{
		Repo:       spec.Repo,
		Ref:        ref,
		SparsePaths: []string{repoRelPath},
	}); err != nil {
		return nil, err
	}

	localPath := filepath.Join(cloneDir, filepath.FromSlash(repoRelPath))
	f.log.Verbosef("  fetched file: %s", localPath)
	return fs.ReadFile(localPath)
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

// isRelativePathStr checks if a path string is relative (not absolute, not tilde)
func isRelativePathStr(p string) bool {
	if p == "" {
		return false
	}
	// Relative paths don't start with / or ~
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") {
		return false
	}
	// Check for relative path patterns: starts with . or contains /..
	return strings.HasPrefix(p, ".") || strings.HasPrefix(p, "./") || strings.Contains(p, "/..")
}

func (f *GitFetcher) String() string {
	return fmt.Sprintf("GitFetcher(cacheDir=%s)", f.cacheDir)
}

