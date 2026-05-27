package qpm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/config"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/fs"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/git"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/graph"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
	"github.com/MatheusQCardoso/homebrew-qpm/internal/swiftpm"
)

type Installer struct {
	packagesDir    string
	git            git.Runner
	log            Logger
	networkTimeout time.Duration
}

func NewInstaller(packagesDir string, log Logger, networkTimeout time.Duration) *Installer {
	if log == nil {
		log = NopLogger{}
	}
	return &Installer{
		packagesDir:    packagesDir,
		git:            git.Runner{Verbose: log.VerboseEnabled()},
		log:            log,
		networkTimeout: networkTimeout,
	}
}

func (i *Installer) InstallAll(ctx context.Context, g *graph.Graph) error {
	if g == nil {
		return fmt.Errorf("graph is required")
	}

	type item struct {
		name  string
		level int
	}
	items := make([]item, 0, len(g.Nodes))
	for name := range g.Nodes {
		lvl, ok := g.Levels[name]
		if !ok {
			lvl = 1_000_000
		}
		items = append(items, item{name: name, level: lvl})
	}
	sort.Slice(items, func(a, b int) bool {
		if items[a].level != items[b].level {
			return items[a].level < items[b].level
		}
		return items[a].name < items[b].name
	})

	localNames := make(map[string]bool, len(g.Nodes))
	for name := range g.Nodes {
		localNames[name] = true
	}

	for _, it := range items {
		spec := g.Nodes[it.name].Normalized()
		if err := spec.Validate(); err != nil {
			return WrapErrorf(err, "%s", it.name)
		}
		i.log.Infof("install %s", it.name)
		i.log.Verbosef("  spec: type=%s repo=%s path=%s ref=%s", spec.Type, spec.Repo, spec.Path, pickRef(spec))
		if err := i.installOne(ctx, it.name, spec, localNames); err != nil {
			return err
		}
		i.log.Infof("installed %s", it.name)
	}

	return nil
}

func (i *Installer) installOne(ctx context.Context, moduleName string, spec model.DepSpec, localNames map[string]bool) error {
	installCtx := ctx
	var cancel context.CancelFunc
	if i.networkTimeout > 0 {
		installCtx, cancel = context.WithTimeout(ctx, i.networkTimeout)
		defer cancel()
	}
	dstDir := filepath.Join(i.packagesDir, moduleName)
	if fs.FileExists(dstDir) {
		if err := fs.RemoveAll(dstDir); err != nil {
			return err
		}
	}

	ref := pickRef(spec)

	switch spec.Type {
	case model.DependencyTypeQPM:
		if spec.Repo == "" {
			i.log.Verbosef("  local qpm root: %s", spec.Path)
			return i.installLocalQPM(moduleName, spec, dstDir)
		}
		manifestRel := qpmPackageJSONRepoPath(moduleName, spec)
		i.log.Verbosef("  sparse clone: repo=%s ref=%s paths=%v", spec.Repo, ref, []string{manifestRel})
		if err := git.SparseClone(installCtx, i.git, dstDir, git.SparseCloneOptions{
			Repo:        spec.Repo,
			Ref:         ref,
			SparsePaths: []string{manifestRel},
		}); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return WrapErrorf(err, "clone %s: timed out after %s", moduleName, i.networkTimeout)
			}
			return WrapErrorf(err, "clone %s", moduleName)
		}

		localManifestPath := filepath.Join(dstDir, filepath.FromSlash(manifestRel))
		rootManifestRel := moduleName + config.QPMManifestFileExtension
		rootManifestPath := filepath.Join(dstDir, rootManifestRel)

		if filepath.ToSlash(manifestRel) != rootManifestRel {
			if err := fs.CopyFile(rootManifestPath, localManifestPath); err != nil {
				return err
			}
			_ = fs.RemoveAll(localManifestPath)
			_ = cleanupEmptyParents(filepath.Dir(localManifestPath), dstDir)
			localManifestPath = rootManifestPath
		}

		b, err := fs.ReadFile(localManifestPath)
		if err != nil {
			return err
		}
		pm, err := model.DecodeStrict[model.QpmPackageManifest](b, moduleName+config.QPMManifestFileExtension)
		if err != nil {
			return err
		}

		pkgSwift, err := swiftpm.RenderPackageSwiftFromQPM(pm)
		if err != nil {
			return err
		}
		if err := fs.WriteFile(filepath.Join(dstDir, config.SPMManifestFileName), pkgSwift); err != nil {
			return err
		}

		targetRelPaths := qpmTargetPaths(pm)
		relocations := make([]relocation, 0, len(targetRelPaths))

		repoPaths := make([]string, 0, 1+len(targetRelPaths))
		repoPaths = append(repoPaths, manifestRel)
		for _, p := range targetRelPaths {
			srcRepoRel := repoRel(spec.Path, p)
			repoPaths = append(repoPaths, srcRepoRel)
			if srcRepoRel != path.Clean(strings.Trim(p, "/")) {
				relocations = append(relocations, relocation{SrcRepoRel: srcRepoRel, DstRel: p})
			}
		}

		i.log.Verbosef("  sparse paths: %v", uniquePosix(repoPaths))
		if err := git.EnsureSparsePaths(installCtx, i.git, dstDir, ref, uniquePosix(repoPaths)); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return WrapErrorf(err, "clone %s: timed out after %s", moduleName, i.networkTimeout)
			}
			return err
		}
		for _, r := range relocations {
			i.log.Verbosef("  relocate: %s -> %s", r.SrcRepoRel, r.DstRel)
			if err := relocateFromRepoRel(dstDir, r.SrcRepoRel, r.DstRel); err != nil {
				return err
			}
		}
		if filepath.ToSlash(manifestRel) != rootManifestRel {
			_ = fs.RemoveAll(filepath.Join(dstDir, filepath.FromSlash(manifestRel)))
			_ = cleanupEmptyParents(filepath.Dir(filepath.Join(dstDir, filepath.FromSlash(manifestRel))), dstDir)
		}

	case model.DependencyTypeSPM:
		if spec.Repo == "" {
			i.log.Verbosef("  local spm root: %s", spec.Path)
			return i.installLocalSPM(moduleName, spec, dstDir, localNames)
		}
		manifestRel := spmPackageSwiftRepoPath(spec)
		i.log.Verbosef("  sparse clone: repo=%s ref=%s paths=%v", spec.Repo, ref, []string{manifestRel})
		if err := git.SparseClone(installCtx, i.git, dstDir, git.SparseCloneOptions{
			Repo:        spec.Repo,
			Ref:         ref,
			SparsePaths: []string{manifestRel},
		}); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return WrapErrorf(err, "clone %s: timed out after %s", moduleName, i.networkTimeout)
			}
			return WrapErrorf(err, "clone %s", moduleName)
		}

		localManifestPath := filepath.Join(dstDir, filepath.FromSlash(manifestRel))
		b, err := fs.ReadFile(localManifestPath)
		if err != nil {
			return err
		}
		manifestContent := string(b)

		rootManifestPath := filepath.Join(dstDir, config.SPMManifestFileName)
		if filepath.ToSlash(manifestRel) != config.SPMManifestFileName {
			if err := fs.WriteFile(rootManifestPath, []byte(manifestContent)); err != nil {
				return err
			}
		} else {
			rootManifestPath = localManifestPath
		}

		targetPaths := swiftpm.ParseSPMTargetPaths(manifestContent)

		repoPaths := make([]string, 0, 1+len(targetPaths))
		repoPaths = append(repoPaths, manifestRel)
		relocations := make([]relocation, 0, len(targetPaths))
		for _, p := range targetPaths {
			srcRepoRel := repoRel(spec.Path, p)
			repoPaths = append(repoPaths, srcRepoRel)
			if srcRepoRel != path.Clean(strings.Trim(p, "/")) {
				relocations = append(relocations, relocation{SrcRepoRel: srcRepoRel, DstRel: p})
			}
		}

		i.log.Verbosef("  sparse paths: %v", uniquePosix(repoPaths))
		if err := git.EnsureSparsePaths(installCtx, i.git, dstDir, ref, uniquePosix(repoPaths)); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return WrapErrorf(err, "clone %s: timed out after %s", moduleName, i.networkTimeout)
			}
			return err
		}
		for _, r := range relocations {
			i.log.Verbosef("  relocate: %s -> %s", r.SrcRepoRel, r.DstRel)
			if err := relocateFromRepoRel(dstDir, r.SrcRepoRel, r.DstRel); err != nil {
				return err
			}
		}

		updated, err := swiftpm.RewriteDependenciesToLocalPaths(manifestContent, localNames)
		if err != nil {
			return err
		}
		if updated != manifestContent {
			if err := fs.WriteFile(rootManifestPath, []byte(updated)); err != nil {
				return err
			}
		}

		if rootManifestPath != localManifestPath {
			_ = fs.RemoveAll(localManifestPath)
			_ = cleanupEmptyParents(filepath.Dir(localManifestPath), dstDir)
		}

	default:
		return fmt.Errorf("%s: unsupported type %q", moduleName, spec.Type)
	}

	_ = fs.RemoveAll(filepath.Join(dstDir, ".git"))

	return nil
}

func (i *Installer) installLocalQPM(moduleName string, spec model.DepSpec, dstDir string) error {
	lp, err := ResolveLocalQpmManifest(moduleName, spec.Path)
	if err != nil {
		return err
	}
	i.log.Verbosef("  local manifest: %s", lp.ManifestPath)
	if err := fs.EnsureDir(dstDir); err != nil {
		return err
	}

	manifestDst := filepath.Join(dstDir, moduleName+config.QPMManifestFileExtension)
	if err := fs.CopyFile(manifestDst, lp.ManifestPath); err != nil {
		return err
	}
	b, err := fs.ReadFile(manifestDst)
	if err != nil {
		return err
	}
	pm, err := model.DecodeStrict[model.QpmPackageManifest](b, moduleName+config.QPMManifestFileExtension)
	if err != nil {
		return err
	}

	pkgSwift, err := swiftpm.RenderPackageSwiftFromQPM(pm)
	if err != nil {
		return err
	}
	if err := fs.WriteFile(filepath.Join(dstDir, config.SPMManifestFileName), pkgSwift); err != nil {
		return err
	}

	for _, p := range qpmTargetPaths(pm) {
		src := filepath.Join(lp.RootDir, filepath.FromSlash(filepath.ToSlash(p)))
		dst := filepath.Join(dstDir, filepath.FromSlash(filepath.ToSlash(p)))
		src = filepath.Clean(src)
		if !filepath.IsAbs(src) {
			var err error
			src, err = filepath.Abs(src)
			if err != nil {
				return err
			}
		}
		i.log.Verbosef("  symlink: %s -> %s", dst, src)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("%s: missing local path %s", moduleName, src)
		}
		if err := fs.Symlink(dst, src); err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) installLocalSPM(moduleName string, spec model.DepSpec, dstDir string, localNames map[string]bool) error {
	lp, err := ResolveLocalSpmManifest(spec.Path)
	if err != nil {
		return err
	}
	i.log.Verbosef("  local manifest: %s", lp.ManifestPath)
	if err := fs.EnsureDir(dstDir); err != nil {
		return err
	}

	manifestContentBytes, err := fs.ReadFile(lp.ManifestPath)
	if err != nil {
		return err
	}
	manifestContent := string(manifestContentBytes)

	updated, err := swiftpm.RewriteDependenciesToLocalPaths(manifestContent, localNames)
	if err != nil {
		return err
	}
	manifestDst := filepath.Join(dstDir, config.SPMManifestFileName)
	if err := fs.WriteFile(manifestDst, []byte(updated)); err != nil {
		return err
	}

	for _, p := range swiftpm.ParseSPMTargetPaths(updated) {
		src := filepath.Join(lp.RootDir, filepath.FromSlash(filepath.ToSlash(p)))
		dst := filepath.Join(dstDir, filepath.FromSlash(filepath.ToSlash(p)))
		src = filepath.Clean(src)
		if !filepath.IsAbs(src) {
			var err error
			src, err = filepath.Abs(src)
			if err != nil {
				return err
			}
		}
		i.log.Verbosef("  symlink: %s -> %s", dst, src)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("%s: missing local path %s", moduleName, src)
		}
		if err := fs.Symlink(dst, src); err != nil {
			return err
		}
	}

	return nil
}

type relocation struct {
	SrcRepoRel string
	DstRel     string
}

func repoRel(base string, rel string) string {
	base = strings.Trim(strings.TrimSpace(base), "/")
	cleanRel := path.Clean(strings.Trim(rel, "/"))
	if base == "" || base == "." {
		return cleanRel
	}
	if strings.HasPrefix(cleanRel, base+"/") || cleanRel == base {
		return cleanRel
	}
	return path.Join(base, cleanRel)
}

func relocateFromRepoRel(moduleDir, srcRepoRel, dstRel string) error {
	srcRepoRel = path.Clean(strings.Trim(srcRepoRel, "/"))
	dstRel = path.Clean(strings.Trim(dstRel, "/"))
	if srcRepoRel == "." || dstRel == "." || srcRepoRel == "" || dstRel == "" {
		return nil
	}
	if srcRepoRel == dstRel {
		return nil
	}

	src := filepath.Join(moduleDir, filepath.FromSlash(srcRepoRel))
	dst := filepath.Join(moduleDir, filepath.FromSlash(dstRel))

	if !fs.FileExists(src) {
		return nil
	}

	_ = fs.RemoveAll(dst)
	if err := fs.EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	if err := os.Rename(src, dst); err == nil {
		return cleanupEmptyParents(filepath.Dir(src), moduleDir)
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := fs.CopyDir(dst, src); err != nil {
			return err
		}
		if err := fs.RemoveAll(src); err != nil {
			return err
		}
		return cleanupEmptyParents(filepath.Dir(src), moduleDir)
	}

	if err := fs.CopyFile(dst, src); err != nil {
		return err
	}
	_ = os.Remove(src)
	return cleanupEmptyParents(filepath.Dir(src), moduleDir)
}

func cleanupEmptyParents(dir, stopAt string) error {
	stopAt = filepath.Clean(stopAt)
	for {
		dir = filepath.Clean(dir)
		if dir == stopAt || dir == "." || dir == string(filepath.Separator) {
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		if len(entries) > 0 {
			return nil
		}

		if err := os.Remove(dir); err != nil {
			return nil
		}
		dir = filepath.Dir(dir)
	}
}

func qpmTargetPaths(pm model.QpmPackageManifest) []string {
	paths := map[string]bool{}
	for _, t := range pm.Package.Targets {
		p := stringsTrimSlash(t.Path)
		if p != "" {
			paths[p] = true
		}
	}
	for _, t := range pm.Package.TestTargets {
		p := stringsTrimSlash(t.Path)
		if p != "" {
			paths[p] = true
		}
	}
	out := make([]string, 0, len(paths))
	for p := range paths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func uniquePosix(in []string) []string {
	set := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = path.Clean(stringsTrimSlash(p))
		if p == "." || p == "" {
			continue
		}
		if !set[p] {
			set[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func stringsTrimSlash(s string) string {
	s = filepath.ToSlash(strings.TrimSpace(s))
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimSuffix(s, "/")
	return s
}
