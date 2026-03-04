package graph

import (
	"context"
	"fmt"
	"sort"

	"github.com/matheusqcardoso/qpm/internal/model"
	"github.com/matheusqcardoso/qpm/internal/swiftpm"
)

type Graph struct {
	Roots  []string                 `json:"roots"`
	Nodes  map[string]model.DepSpec `json:"nodes"`
	Edges  map[string][]string      `json:"edges"`
	Levels map[string]int           `json:"levels"`
}

type ManifestFetcher interface {
	FetchQpmPackageJSON(ctx context.Context, moduleName string, spec model.DepSpec) ([]byte, error)
	FetchSpmPackageSwift(ctx context.Context, moduleName string, spec model.DepSpec) ([]byte, error)
}

func Build(ctx context.Context, m model.QuirinoManifest, fetcher ManifestFetcher) (*Graph, error) {
	if fetcher == nil {
		return nil, fmt.Errorf("fetcher is required")
	}
	if len(m.Dependencies) == 0 {
		return &Graph{
			Roots:  []string{},
			Nodes:  map[string]model.DepSpec{},
			Edges:  map[string][]string{},
			Levels: map[string]int{},
		}, nil
	}

	g := &Graph{
		Roots:  make([]string, 0, len(m.Dependencies)),
		Nodes:  map[string]model.DepSpec{},
		Edges:  map[string][]string{},
		Levels: map[string]int{},
	}

	for name, spec := range m.Dependencies {
		ns := spec.Normalized()
		if err := ns.Validate(); err != nil {
			return nil, fmt.Errorf("top-level dependency %s: %w", name, err)
		}
		g.Roots = append(g.Roots, name)
		g.Nodes[name] = ns
	}
	sort.Strings(g.Roots)

	queue := make([]string, 0, len(g.Roots))
	queue = append(queue, g.Roots...)
	seen := map[string]bool{}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true

		spec, ok := g.Nodes[name]
		if !ok {
			continue
		}

		switch spec.Type {
		case model.DependencyTypeQPM:
			b, err := fetcher.FetchQpmPackageJSON(ctx, name, spec)
			if err != nil {
				id := spec.Repo
				if id == "" {
					id = spec.Path
				}
				return nil, fmt.Errorf("fetch %s (%s): %w", name, id, err)
			}
			pm, err := model.DecodeStrict[model.QpmPackageManifest](b, name+".Package.json")
			if err != nil {
				return nil, err
			}
			for depName, depSpec := range pm.Package.Dependencies {
				ds := depSpec.Normalized()
				if err := ds.Validate(); err != nil {
					return nil, fmt.Errorf("%s dependency %s: %w", name, depName, err)
				}
				g.Edges[name] = appendUnique(g.Edges[name], depName)
				if _, exists := g.Nodes[depName]; !exists {
					g.Nodes[depName] = ds
					queue = append(queue, depName)
				} else {
					// Rule 2.1: keep uppermost spec already in Nodes.
					queue = append(queue, depName)
				}
			}

		case model.DependencyTypeSPM:
			b, err := fetcher.FetchSpmPackageSwift(ctx, name, spec)
			if err != nil {
				id := spec.Repo
				if id == "" {
					id = spec.Path
				}
				return nil, fmt.Errorf("fetch %s (%s): %w", name, id, err)
			}
			deps, err := swiftpm.ParseSPMDependencies(string(b))
			if err != nil {
				return nil, err
			}
			for _, d := range deps {
				g.Edges[name] = appendUnique(g.Edges[name], d.Name)
				if _, exists := g.Nodes[d.Name]; !exists {
					g.Nodes[d.Name] = d.Spec.Normalized()
					queue = append(queue, d.Name)
				} else {
					queue = append(queue, d.Name)
				}
			}
		default:
			return nil, fmt.Errorf("unsupported type for %s: %q", name, spec.Type)
		}
	}

	// Deterministic edges.
	for from, tos := range g.Edges {
		sort.Strings(tos)
		g.Edges[from] = uniqueSorted(tos)
	}

	g.Levels = computeLevels(g.Roots, g.Edges)
	return g, nil
}

func computeLevels(roots []string, edges map[string][]string) map[string]int {
	levels := map[string]int{}
	type item struct {
		name  string
		level int
	}
	q := make([]item, 0, len(roots))
	for _, r := range roots {
		q = append(q, item{name: r, level: 0})
	}
	for len(q) > 0 {
		it := q[0]
		q = q[1:]
		if prev, ok := levels[it.name]; ok && prev <= it.level {
			continue
		}
		levels[it.name] = it.level
		for _, dep := range edges[it.name] {
			q = append(q, item{name: dep, level: it.level + 1})
		}
	}
	return levels
}

func appendUnique(list []string, v string) []string {
	for _, s := range list {
		if s == v {
			return list
		}
	}
	return append(list, v)
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return in
	}
	out := make([]string, 0, len(in))
	prev := ""
	for i, s := range in {
		if i == 0 || s != prev {
			out = append(out, s)
		}
		prev = s
	}
	return out
}

