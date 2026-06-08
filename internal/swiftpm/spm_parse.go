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

func ParseSPMDependencies(src string) ([]SwiftDependency, error) {
	var deps []SwiftDependency
	stringEnv := collectSwiftStringConstants(src)

	packageBlocks := extractCallBlocks(src, ".package(")
	for _, block := range packageBlocks {
		if strings.Contains(block, "path:") && !strings.Contains(block, "url:") {
			continue
		}

		url := expandSwiftInterpolations(firstStringArg(block, "url:"), stringEnv)
		if url == "" {
			continue
		}

		name := expandSwiftInterpolations(firstStringArg(block, "name:"), stringEnv)
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

		switch {
		case strings.Contains(block, "exact:"):
			spec.Tag = expandSwiftInterpolations(firstStringArg(block, "exact:"), stringEnv)
		case strings.Contains(block, ".exact("):
			spec.Tag = expandSwiftInterpolations(firstCallStringArg(block, ".exact("), stringEnv)
		case strings.Contains(block, "branch:"):
			spec.Branch = expandSwiftInterpolations(firstStringArg(block, "branch:"), stringEnv)
		case strings.Contains(block, ".branch("):
			spec.Branch = expandSwiftInterpolations(firstCallStringArg(block, ".branch("), stringEnv)
		case strings.Contains(block, "revision:"):
			spec.Revision = expandSwiftInterpolations(firstStringArg(block, "revision:"), stringEnv)
		case strings.Contains(block, ".revision("):
			spec.Revision = expandSwiftInterpolations(firstCallStringArg(block, ".revision("), stringEnv)
		case strings.Contains(block, "from:"):
			spec.Tag = expandSwiftInterpolations(firstStringArg(block, "from:"), stringEnv)
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

	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	return deps, nil
}

func ParseSPMTargetPaths(src string) []string {
	targetBlocks := append(extractCallBlocks(src, ".target("), extractCallBlocks(src, ".testTarget(")...)
	paths := map[string]bool{}
	for _, block := range targetBlocks {
		pathValue := firstStringArg(block, "path:")
		if pathValue == "" {
			continue
		}
		pathValue = strings.Trim(pathValue, "/")
		if pathValue != "" {
			paths[pathValue] = true
		}
	}
	if len(paths) == 0 {
		return []string{"Sources", "Tests"}
	}
	out := make([]string, 0, len(paths))
	for pathValue := range paths {
		out = append(out, pathValue)
	}
	sort.Strings(out)
	return out
}

func RewriteDependenciesToLocalPaths(src string, localNames map[string]bool) (string, error) {
	if len(localNames) == 0 {
		return src, nil
	}
	stringEnv := collectSwiftStringConstants(src)

	packageSpans := extractCallBlocksWithSpans(src, ".package(")
	if len(packageSpans) == 0 {
		return src, nil
	}

	var builder strings.Builder
	last := 0
	for _, span := range packageSpans {
		builder.WriteString(src[last:span.Start])

		content := src[span.Start:span.End]
		url := expandSwiftInterpolations(firstStringArg(content, "url:"), stringEnv)
		if url == "" {
			builder.WriteString(content)
			last = span.End
			continue
		}

		name := expandSwiftInterpolations(firstStringArg(content, "name:"), stringEnv)
		if name == "" {
			name = guessNameFromURL(url)
		}

		if name != "" && localNames[name] {
			indent := detectIndentBefore(src, span.Start)
			builder.WriteString(indent)
			builder.WriteString(".package(path: \"../")
			builder.WriteString(name)
			builder.WriteString("\")")
		} else {
			builder.WriteString(content)
		}

		last = span.End
	}
	builder.WriteString(src[last:])
	return builder.String(), nil
}

type span struct {
	Start int
	End   int
}

func extractCallBlocks(src, needle string) []string {
	callSpans := extractCallBlocksWithSpans(src, needle)
	blocks := make([]string, 0, len(callSpans))
	for _, span := range callSpans {
		blocks = append(blocks, src[span.Start:span.End])
	}
	return blocks
}

func extractCallBlocksWithSpans(src, needle string) []span {
	var callSpans []span
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
		callSpans = append(callSpans, span{Start: start, End: end})
		i = end
	}
	return callSpans
}

func findMatchingParenEnd(src string, openParenIdx int) int {
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
	regex, ok := reStringArgCache[key]
	if !ok {
		regex = regexp.MustCompile(regexp.QuoteMeta(key) + `\s*\"([^\"]+)\"`)
		reStringArgCache[key] = regex
	}
	match := regex.FindStringSubmatch(block)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

var reCallStringArgCache = map[string]*regexp.Regexp{}

func firstCallStringArg(block, call string) string {
	regex, ok := reCallStringArgCache[call]
	if !ok {
		regex = regexp.MustCompile(regexp.QuoteMeta(call) + `\s*\"([^\"]+)\"`)
		reCallStringArgCache[call] = regex
	}
	match := regex.FindStringSubmatch(block)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func guessNameFromURL(url string) string {
	trimmedURL := strings.TrimSpace(url)
	trimmedURL = strings.TrimSuffix(trimmedURL, "/")
	base := path.Base(trimmedURL)
	base = strings.TrimSuffix(base, ".git")
	return base
}

func detectIndentBefore(src string, idx int) string {
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

// reSwiftLetString matches top-level string bindings in Package.swift so that
// dependency URLs using Swift interpolation (e.g. "\(internalGit)/my-repo")
// can be resolved after extraction by firstStringArg.
var reSwiftLetString = regexp.MustCompile(
	`(?m)^[ \t]*(?:(?:private|fileprivate|internal|public|open)[ \t]+)?let[ \t]+(\w+)[ \t]*=[ \t]*"([^"]*)"[ \t;]*(?:[ \t]*//[^\n]*)?$`,
)

func collectSwiftStringConstants(src string) map[string]string {
	out := make(map[string]string)
	for _, m := range reSwiftLetString.FindAllStringSubmatch(src, -1) {
		if len(m) == 3 {
			out[m[1]] = m[2]
		}
	}
	return out
}

func expandSwiftInterpolations(s string, env map[string]string) string {
	if len(env) == 0 || !strings.Contains(s, `\(`) {
		return s
	}
	const maxPasses = 64
	for pass := 0; pass < maxPasses; pass++ {
		changed := false
		for start := 0; start < len(s)-2; start++ {
			if s[start] != '\\' || s[start+1] != '(' {
				continue
			}
			j := start + 2
			if j >= len(s) || !isSwiftIdentByte(s[j], true) {
				continue
			}
			j++
			for j < len(s) && isSwiftIdentByte(s[j], false) {
				j++
			}
			if j >= len(s) || s[j] != ')' {
				continue
			}
			name := s[start+2 : j]
			repl, ok := env[name]
			if !ok {
				continue
			}
			s = s[:start] + repl + s[j+1:]
			changed = true
			break
		}
		if !changed {
			break
		}
	}
	return s
}

func isSwiftIdentByte(c byte, first bool) bool {
	if first {
		return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
	}
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (c >= '0' && c <= '9')
}
