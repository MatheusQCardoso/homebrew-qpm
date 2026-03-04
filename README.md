# QPM (Quirino's Package Manager)

`qpm` reads a `Quirino.json` in your project root, resolves all transitive dependencies, and materializes them into a sibling `QPackages/` directory using **sparse git checkouts** (manifest first, then only the referenced target source/test paths).

It also supports **local dependencies** (omit `repo`, provide an absolute `path`), which are materialized by copying the manifest into `QPackages/<ModuleName>/` and **symlinking** the referenced target paths from your local package.

## Install

From `qpmclitool/`:

```bash
go build -o qpm ./cmd/qpm
```

## Use

From a project folder that contains `Quirino.json`:

```bash
./path/to/qpm install
```

This will:

- Create `QPackages/` next to `Quirino.json`
- Create/overwrite `QPackages/<ModuleName>` for every resolved module
- Write `QPackages/qpm.lock.json` with the resolved dependency graph

## Commands

- `qpm install`: build graph + materialize `QPackages/`
- `qpm graph`: print the resolved dependency graph JSON
- `qpm clean`: delete `QPackages/` (including `qpm.lock.json` and cache)
