package swiftpm

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
)

type SwiftDependency struct {
	Name string
	Spec model.DepSpec
}

// ParseSPMDependencies extracts `.package(...)` remote dependencies from a Package.swift.
// This is a best-effort parser intended for common SwiftPM manifests.
func ParseSPMDependencies(src string) ([]SwiftDependency, error) {
	var deps []SwiftDependency

	blocks := extractCallBlocks(src, ".package(")
	for _, b := range blocks {
		if strings.Contains(b, "path:") && !strings.Contains(b, "url:") {
			continue // already local
		}

		url := firstStringArg(b, "url:")
		if url == "" {
			continue
		}

		name := firstStringArg(b, "name:")
		if name == "" {
			name = guessNameFromURL(url)
		}
		if name == "" {
			continue
		}

		spec := model.DepSpec{
			Type: model.DependencyTypeSPM,
			Repo: url,
		}

		// Support both labeled and unlabeled requirement syntaxes:
		// - exact: "1.2.3"  / .exact("1.2.3")
		// - branch: "main"  / .branch("main")
		// - revision: "sha" / .revision("sha")
		switch {
		case strings.Contains(b, "exact:"):
			spec.Tag = firstStringArg(b, "exact:")
		case strings.Contains(b, ".exact("):
			spec.Tag = firstCallStringArg(b, ".exact(")
		case strings.Contains(b, "branch:"):
			spec.Branch = firstStringArg(b, "branch:")
		case strings.Contains(b, ".branch("):
			spec.Branch = firstCallStringArg(b, ".branch(")
		case strings.Contains(b, "revision:"):
			spec.Revision = firstStringArg(b, "revision:")
		case strings.Contains(b, ".revision("):
			spec.Revision = firstCallStringArg(b, ".revision(")
		case strings.Contains(b, "from:"):
			// Not "exact", but we still need a concrete tag to checkout.
			spec.Tag = firstStringArg(b, "from:")
		}

		spec = spec.Normalized()
		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf("spm dep %s: %w", name, err)
		}

		deps = append(deps, SwiftDependency{
			Name: name,
			Spec: spec,
		})
	}

	// Deterministic order.
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	return deps, nil
}

// ParseSPMTargetPaths extracts target/testTarget `path:` values. If no explicit paths
// exist, it returns ["Sources", "Tests"].
func ParseSPMTargetPaths(src string) []string {
	blocks := append(extractCallBlocks(src, ".target("), extractCallBlocks(src, ".testTarget(")...)
	paths := map[string]bool{}
	for _, b := range blocks {
		p := firstStringArg(b, "path:")
		if p == "" {
			continue
		}
		p = strings.Trim(p, "/")
		if p != "" {
			paths[p] = true
		}
	}
	if len(paths) == 0 {
		return []string{"Sources", "Tests"}
	}
	out := make([]string, 0, len(paths))
	for p := range paths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// RewriteDependenciesToLocalPaths replaces remote `.package(...)` blocks whose `name:`
// (or URL-inferred name) matches localNames with `.package(path: "../Name")`.
// It preserves the rest of the manifest.
func RewriteDependenciesToLocalPaths(src string, localNames map[string]bool) (string, error) {
	if len(localNames) == 0 {
		return src, nil
	}

	blocks := extractCallBlocksWithSpans(src, ".package(")
	if len(blocks) == 0 {
		return src, nil
	}

	var b strings.Builder
	last := 0
	for _, blk := range blocks {
		b.WriteString(src[last:blk.Start])

		content := src[blk.Start:blk.End]
		url := firstStringArg(content, "url:")
		if url == "" {
			// Keep local packages or weird entries untouched.
			b.WriteString(content)
			last = blk.End
			continue
		}

		name := firstStringArg(content, "name:")
		if name == "" {
			name = guessNameFromURL(url)
		}

		if name != "" && localNames[name] {
			indent := detectIndentBefore(src, blk.Start)
			b.WriteString(indent)
			b.WriteString(".package(path: \"../")
			b.WriteString(name)
			b.WriteString("\")")
		} else {
			b.WriteString(content)
		}

		last = blk.End
	}
	b.WriteString(src[last:])
	return b.String(), nil
}

type span struct {
	Start int
	End   int
}

func extractCallBlocks(src, needle string) []string {
	spans := extractCallBlocksWithSpans(src, needle)
	out := make([]string, 0, len(spans))
	for _, s := range spans {
		out = append(out, src[s.Start:s.End])
	}
	return out
}

func extractCallBlocksWithSpans(src, needle string) []span {
	var spans []span
	for i := 0; i < len(src); {
		idx := strings.Index(src[i:], needle)
		if idx < 0 {
			break
		}
		start := i + idx
		end := findMatchingParenEnd(src, start+len(needle)-1)
		if end <= start {
			i = start + len(needle)
			continue
		}
		spans = append(spans, span{Start: start, End: end})
		i = end
	}
	return spans
}

func findMatchingParenEnd(src string, openParenIdx int) int {
	// openParenIdx points at the '('.
	depth := 0
	inString := false
	escaped := false
	for i := openParenIdx; i < len(src); i++ {
		ch := src[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

var reStringArgCache = map[string]*regexp.Regexp{}

func firstStringArg(block, key string) string {
	re, ok := reStringArgCache[key]
	if !ok {
		re = regexp.MustCompile(regexp.QuoteMeta(key) + `\s*\"([^\"]+)\"`)
		reStringArgCache[key] = re
	}
	m := re.FindStringSubmatch(block)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

var reCallStringArgCache = map[string]*regexp.Regexp{}

func firstCallStringArg(block, call string) string {
	re, ok := reCallStringArgCache[call]
	if !ok {
		re = regexp.MustCompile(regexp.QuoteMeta(call) + `\s*\"([^\"]+)\"`)
		reCallStringArgCache[call] = re
	}
	m := re.FindStringSubmatch(block)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func guessNameFromURL(url string) string {
	u := strings.TrimSpace(url)
	u = strings.TrimSuffix(u, "/")
	base := path.Base(u)
	base = strings.TrimSuffix(base, ".git")
	return base
}

func detectIndentBefore(src string, idx int) string {
	// Find line start.
	lineStart := strings.LastIndex(src[:idx], "\n")
	if lineStart < 0 {
		lineStart = 0
	} else {
		lineStart++
	}
	indent := src[lineStart:idx]
	indent = strings.TrimRight(indent, "\r")
	return indent
}

