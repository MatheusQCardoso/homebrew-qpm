# QPM (Quirino's Package Manager)

`qpm` reads a `Quirino.json` in your project root, resolves all transitive dependencies, and materializes them into a sibling `QPackages/` directory using **sparse git checkouts** (manifest first, then only the referenced target source/test paths).

It also supports **local dependencies** (omit `repo`, provide an absolute `path`), which are materialized by copying the manifest into `QPackages/<ModuleName>/` and **symlinking** the referenced target paths from your local package.

---

## Install

**Compile** `qpm` **yourself** by running

```bash
go build -o qpm ./cmd/qpm
```

*Must have Golang (`brew install go`)*

OR, alternatively, **install from HomeBrew**:

```bash
brew tap MatheusQCardoso/qpm
brew install qpm
sudo xattr -d com.apple.quarantine $(which qpm)
```

*Since the binaries are currently unsigned ~~($99/yr)~~, the last line ensures Apple won't complain when you try to use the binary that was downloaded from Homebrew*

---

## Why use QPM (vs “normal” SwiftPM downloads/clones)?

When you add remote dependencies directly in SwiftPM, Xcode/SwiftPM typically ends up cloning whole repositories (and keeping `.git` metadata) for every dependency version it needs. In monorepos this can be especially wasteful.

QPM is optimized for **fast, minimal checkouts** and a predictable on-disk layout:

- **Sparse checkouts (manifest-first, sources second)**: QPM fetches only the package manifest first, then checks out only the exact target/test paths referenced by the manifest.
- **Monorepo-friendly `path` prefixing**: Packages can live under a subdirectory while still materializing into `QPackages/<Name>/`.
- **Unified “local packages” story**: Local dependencies are materialized into `QPackages/` and referenced as local `../<Name>` packages (via generated/rewritten manifests), keeping consumers consistent.
- **Deterministic resolution output**: QPM writes `QPackages/qpm.lock.json` containing the resolved dependency graph for debugging/reproducibility.
- **Works with both QPM and SwiftPM packages**:
  - QPM packages are described by `<ModuleName>.Package.json` and rendered into a `Package.swift`.
  - SwiftPM packages are discovered from `Package.swift` and best-effort rewritten to point at local siblings.

---

## Quick start

From a project folder that contains `Quirino.json`:

```bash
qpm install
```

This will:

- Create `<packages-dir>/` next to `Quirino.json` (default: `QPackages/`)
- Create/overwrite `<packages-dir>/<ModuleName>/` for every resolved module
- Write `<packages-dir>/qpm.lock.json` (resolved dependency graph)

---

## Command-line tool

```bash
qpm <command> [options]
```

### Commands

- **`qpm install`**: Resolve deps and materialize `QPackages/`
- **`qpm graph`**: Print the resolved dependency graph (JSON)
- **`qpm clean`**: Delete `QPackages/` and qpm-generated leftovers
- **`qpm help` / `qpm --help` / `qpm -h`**: Show help

### Options

#### Common (all commands)

- **`--file <path>`**: Path to `Quirino.json` (default: `Quirino.json`)
- **`--packages-dir <name>`**: Output directory name created next to `Quirino.json` (default: `QPackages`)
  - Must be a **directory name only** (no path separators, not absolute). QPM enforces that it sits **next to** `Quirino.json`.

#### `install` only

- **`--verbose`**: Verbose output

### Exit codes

- **0**: success (or help shown)
- **1**: runtime error
- **2**: CLI/flag parsing error (e.g. unknown command, invalid flags)

--- 

## `Quirino.json` (project manifest) reference

`Quirino.json` lives in your project root (or you point `--file` to it). It declares top-level dependencies and is also used to compute where `<packages-dir>/` is created.

### Shape

```json
{
  "dependencies": {
    "<ModuleName>": { /* DepSpec */ }
  }
}
```

`<ModuleName>` is **significant**:

- It becomes the folder name `QPackages/<ModuleName>/`.
- For QPM modules, it also determines the required manifest filename `<ModuleName>.Package.json`.

### DepSpec keys (per module)

All keys below are supported. `Quirino.json` is decoded as JSON (and must not contain trailing extra JSON content).

- **`type`** *(optional)*: `"qpm"` or `"spm"` (default: `"qpm"`)
  - `"qpm"` means the package is described by `<ModuleName>.Package.json` and QPM will generate a `Package.swift`.
  - `"spm"` means the package is described by `Package.swift` and QPM will parse it best-effort.
- **`repo`** *(optional)*: remote git URL (when present, dependency is treated as **remote**)
- **`path`** *(optional)*:
  - For **remote** deps: repo-relative subdirectory prefix (monorepo support).
    - QPM: where to find `<ModuleName>.Package.json`
    - SPM: where to find `Package.swift`
  - For **local** deps (when `repo` is omitted): absolute path (or `~`-prefixed) where QPM will search for the manifest.
- **`branch`** *(optional)*: git ref to checkout
- **`tag`** *(optional)*: git ref to checkout (also used for SwiftPM’s `.exact(...)` and `.from(...)` parsing)
- **`revision`** *(optional)*: git SHA to checkout

#### Ref rule (important)

At most **one** of `branch`, `tag`, or `revision` may be set.

#### Local vs remote rule (important)

- **Remote dependency**: `repo` is set.
- **Local dependency**: `repo` is omitted and `path` is required.
  - Local `path` must be **absolute** or start with `~`.
  - Local dependencies cannot specify `branch`/`tag`/`revision`.

### Examples

#### Remote QPM module (repo root)

```json
{
  "dependencies": {
    "MyModule": {
      "type": "qpm",
      "repo": "https://github.com/acme/my-module",
      "tag": "1.2.3"
    }
  }
}
```

This expects `MyModule.Package.json` at the repo root.

#### Remote QPM module in a monorepo subdir

```json
{
  "dependencies": {
    "MyModule": {
      "type": "qpm",
      "repo": "https://github.com/acme/monorepo",
      "branch": "main",
      "path": "packages/MyModule"
    }
  }
}
```

This expects `packages/MyModule/MyModule.Package.json` in the repo, but QPM will still materialize it as `QPackages/MyModule/`.

#### Remote SwiftPM module (repo root)

```json
{
  "dependencies": {
    "Alamofire": {
      "type": "spm",
      "repo": "https://github.com/Alamofire/Alamofire",
      "tag": "5.10.2"
    }
  }
}
```

This expects `Package.swift` at the repo root.

#### Local QPM module

```json
{
  "dependencies": {
    "MyModule": {
      "type": "qpm",
      "path": "/Users/me/work/monorepo"
    }
  }
}
```

QPM will search (recursively) under the given path for `MyModule.Package.json` (skipping `.git`, `.build`, `DerivedData`, `node_modules`).

#### Local SwiftPM module

```json
{
  "dependencies": {
    "SomeSPM": {
      "type": "spm",
      "path": "~/work/some-spm-repo"
    }
  }
}
```

QPM will search (recursively) under the given path for `Package.swift` (same skip rules as above).

---

## `<ModuleName>.Package.json` (QPM package manifest) reference

QPM packages are described by a JSON manifest named **exactly** `<ModuleName>.Package.json`. This file can live in the repository root or under the repo subdirectory configured by DepSpec `path`.

### Top-level keys

- **`swift-tools-version`** *(required)*: string (rendered into `// swift-tools-version: X.Y`)
- **`version`** *(optional)*: string (rendered into `let version = "..."`)
- **`package`** *(required)*: object with the keys below

### `package` keys

- **`name`** *(required)*: string (SwiftPM `Package(name: ...)`)
- **`min-ios-version`** *(optional)*: string (e.g. `"15"` or `"15.0"`)
  - Only the **major** version is used; invalid/missing values default to **15**.
- **`products`** *(optional but typically present)*: array of products
  - Supported product types: **only** `"library"`.
- **`dependencies`** *(optional)*: object map of dependencies by name
  - Values are the same DepSpec object as in `Quirino.json`.
- **`targets`** *(optional)*: object map of targets by target name
- **`testTargets`** *(optional)*: object map of test targets by target name

### Product keys (`package.products[]`)

- **`type`** *(required)*: must be `"library"`
- **`name`** *(required)*: string
- **`targets`** *(required)*: array of strings (target names)

### Target keys (`package.targets.<TargetName>`)

- **`path`** *(required)*: string (relative path that will be checked out/materialized)
- **`dependencies`** *(optional)*: array of strings
  - If the string matches a **local target name**, it is rendered as `.target(name: "...")`.
  - Otherwise it is rendered as `.product(name: "<Dep>", package: "<Dep>")`.

### Test target keys (`package.testTargets.<TargetName>`)

Same shape as targets:

- **`path`** *(required)*
- **`dependencies`** *(optional)*: array of strings

---

## How it works

### 1) Dependency Graph

`qpm install` and `qpm graph` both start by decoding `Quirino.json` and building a transitive dependency graph:

- **QPM modules**: fetch `<ModuleName>.Package.json`, decode it, and enqueue its `package.dependencies`.
- **SwiftPM modules**: fetch `Package.swift`, parse `.package(...)` blocks best-effort, and enqueue discovered dependencies.

Graph building is manifest-only: it does not clone full source trees up front.

### 2) Manifest fetch via sparse git (or local lookup)

For **remote** dependencies, QPM uses a sparse checkout containing only:

- the manifest file (`<ModuleName>.Package.json` for QPM; `Package.swift` for SwiftPM)

For **local** dependencies, QPM finds manifests by:

- direct child lookup (`<basePath>/<manifest>`)
- otherwise walking the directory tree (skipping `.git`, `.build`, `DerivedData`, `node_modules`)

### 3) Install each module into `QPackages/<Name>/`

Install order is deterministic (based on computed dependency “levels” + name).

#### QPM install flow

For each QPM module:

- sparse-clone the repo (manifest only)
- decode `<Name>.Package.json`
- generate `QPackages/<Name>/Package.swift` from the JSON manifest
- compute the list of target/test paths from the JSON manifest
- extend the sparse checkout to include those paths (plus the manifest)
- if the module was configured with a monorepo `path` prefix, QPM will **relocate** checked-out directories so that the final on-disk layout matches the `path` values referenced by the generated `Package.swift`
- remove `QPackages/<Name>/.git` (only checkout results remain)

For **local** QPM modules (no `repo`, absolute `path`):

- locate `<Name>.Package.json` under the provided `path`
- copy it into `QPackages/<Name>/<Name>.Package.json`
- generate `QPackages/<Name>/Package.swift`
- **symlink** each referenced target/test `path` from the local package into `QPackages/<Name>/...`

#### SwiftPM install flow

For each SwiftPM module:

- sparse-clone the repo (manifest only) and ensure the resulting `QPackages/<Name>/Package.swift` lives at the module root
- determine target/test paths by extracting explicit `path:` values from `.target(...)` / `.testTarget(...)`
  - if no explicit paths exist, QPM assumes `Sources` and `Tests`
- extend the sparse checkout to include those paths
- relocate directories when a monorepo `path` prefix is in play (same idea as QPM packages)
- best-effort rewrite dependency entries so that packages that are also present in `QPackages/` are referenced as:
  - `.package(path: "../<DepName>")`
- remove `QPackages/<Name>/.git`

For **local** SwiftPM modules (no `repo`, absolute `path`):

- locate `Package.swift` under the provided `path`
- write `QPackages/<Name>/Package.swift` (after best-effort dependency rewriting)
- **symlink** each referenced target/test `path` from the local package into `QPackages/<Name>/...`

### 4) Cache and lockfile

- **Manifest fetch cache**: QPM uses `<packages-dir>/.qpm-cache/` as a temp/cache area during manifest fetches.
- **Lockfile**: `qpm install` writes `<packages-dir>/qpm.lock.json` (the resolved dependency graph: roots, nodes/specs, edges, and computed levels).

### 5) Cleanup

`qpm clean` removes the entire `<packages-dir>/` directory, including:

- all materialized modules
- `.qpm-cache/`
- `qpm.lock.json`
