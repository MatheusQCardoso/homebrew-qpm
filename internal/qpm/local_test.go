package qpm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
)

func TestResolveLocalQpmManifestReturnsAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(dir, "project_a")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(projectDir, "ModuleA.Package.json")
	if err := os.WriteFile(manifestPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	projectDir, err = filepath.EvalSymlinks(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath, err = filepath.EvalSymlinks(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	lp, err := ResolveLocalQpmManifest("ModuleA", "project_a")
	if err != nil {
		t.Fatal(err)
	}

	if !filepath.IsAbs(lp.RootDir) {
		t.Fatalf("expected absolute RootDir, got %q", lp.RootDir)
	}
	if !filepath.IsAbs(lp.ManifestPath) {
		t.Fatalf("expected absolute ManifestPath, got %q", lp.ManifestPath)
	}
	if lp.RootDir != projectDir {
		t.Fatalf("unexpected root dir: got %q, want %q", lp.RootDir, projectDir)
	}
	if lp.ManifestPath != manifestPath {
		t.Fatalf("unexpected manifest path: got %q, want %q", lp.ManifestPath, manifestPath)
	}
}

func TestInstallLocalQPMCreatesAbsoluteSymlinkTargets(t *testing.T) {
	dir := t.TempDir()
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(dir, "package_root")
	sourceDir := filepath.Join(packageRoot, "project_a", "Sources")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(packageRoot, "ModuleA.Package.json")
	manifestContent := `{
  "swift-tools-version": "5.9",
  "version": "1.0.0",
  "package": {
    "name": "ModuleA",
    "min-ios-version": "16.0",
    "products": [],
    "dependencies": {},
    "targets": {
      "ModuleA": { "path": "project_a/Sources", "dependencies": [] }
    },
    "testTargets": {}
  }
}`
	if err := os.WriteFile(manifest, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}
	packageRoot, err = filepath.EvalSymlinks(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	sourceDir, err = filepath.EvalSymlinks(sourceDir)
	if err != nil {
		t.Fatal(err)
	}

	dstDir := filepath.Join(dir, "QPackages", "ModuleA")
	installer := NewInstaller(filepath.Join(dir, "QPackages"), NopLogger{})
	spec := model.DepSpec{Type: model.DependencyTypeQPM, Path: packageRoot}
	if err := installer.installLocalQPM("ModuleA", spec, dstDir); err != nil {
		t.Fatal(err)
	}

	tsymlink := filepath.Join(dstDir, "project_a", "Sources")
	info, err := os.Lstat(tsymlink)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink at %s", tsymlink)
	}
	target, err := os.Readlink(tsymlink)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(target) {
		t.Fatalf("expected absolute symlink target, got %q", target)
	}
	if target != sourceDir {
		t.Fatalf("unexpected symlink target: got %q, want %q", target, sourceDir)
	}
}
