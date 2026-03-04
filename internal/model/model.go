package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type DependencyType string

const (
	DependencyTypeQPM DependencyType = "qpm"
	DependencyTypeSPM DependencyType = "spm"
)

type Ref struct {
	Branch   string `json:"branch,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Revision string `json:"revision,omitempty"`
}

func (r Ref) String() string {
	switch {
	case r.Branch != "":
		return "branch:" + r.Branch
	case r.Tag != "":
		return "tag:" + r.Tag
	case r.Revision != "":
		return "revision:" + r.Revision
	default:
		return ""
	}
}

func (r Ref) Validate() error {
	set := 0
	if r.Branch != "" {
		set++
	}
	if r.Tag != "" {
		set++
	}
	if r.Revision != "" {
		set++
	}
	if set > 1 {
		return fmt.Errorf("only one of branch/tag/revision may be set (got %s)", r.String())
	}
	return nil
}

type DepSpec struct {
	Type DependencyType `json:"type,omitempty"`
	Repo string         `json:"repo,omitempty"`
	Ref
	Path string `json:"path,omitempty"`
}

func (s DepSpec) Normalized() DepSpec {
	out := s
	out.Type = DependencyType(strings.ToLower(string(out.Type)))
	out.Repo = strings.TrimSpace(out.Repo)
	p := strings.TrimSpace(out.Path)
	// Keep absolute/tilde paths intact for local dependencies.
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") {
		p = strings.TrimRight(p, "/")
	} else {
		p = strings.Trim(p, "/")
	}
	out.Path = p
	out.Branch = strings.TrimSpace(out.Branch)
	out.Tag = strings.TrimSpace(out.Tag)
	out.Revision = strings.TrimSpace(out.Revision)
	if out.Type == "" {
		out.Type = DependencyTypeQPM
	}
	return out
}

func (s DepSpec) Validate() error {
	s = s.Normalized()
	if err := s.Ref.Validate(); err != nil {
		return err
	}
	switch s.Type {
	case DependencyTypeQPM, DependencyTypeSPM:
	default:
		return fmt.Errorf("unsupported type %q (supported: %q, %q)", s.Type, DependencyTypeQPM, DependencyTypeSPM)
	}

	// Remote dependency (default): requires repo.
	if s.Repo != "" {
		return nil
	}

	// Local dependency: requires an absolute (or ~) path and no ref.
	if s.Path == "" {
		return fmt.Errorf("either repo (remote) or path (local) is required")
	}
	if s.Branch != "" || s.Tag != "" || s.Revision != "" {
		return fmt.Errorf("local dependency cannot specify branch/tag/revision")
	}
	if !strings.HasPrefix(s.Path, "/") && !strings.HasPrefix(s.Path, "~") {
		return fmt.Errorf("local dependency path must be absolute (or start with ~): %q", s.Path)
	}
	return nil
}

type QuirinoManifest struct {
	Dependencies map[string]DepSpec `json:"dependencies"`
}

type QpmPackageManifest struct {
	SwiftToolsVersion string                   `json:"swift-tools-version"`
	Version           string                   `json:"version"`
	Package           QpmPackageDeclaration    `json:"package"`
}

type QpmPackageDeclaration struct {
	Name          string                    `json:"name"`
	MinIOSVersion string                    `json:"min-ios-version"`
	Products      []QpmProduct              `json:"products"`
	Dependencies  map[string]DepSpec        `json:"dependencies"`
	Targets       map[string]QpmTarget      `json:"targets"`
	TestTargets   map[string]QpmTestTarget  `json:"testTargets"`
}

type QpmProduct struct {
	Type    string   `json:"type"` // library
	Name    string   `json:"name"`
	Targets []string `json:"targets"`
}

type QpmTarget struct {
	Path         string   `json:"path"`
	Dependencies []string `json:"dependencies"`
}

type QpmTestTarget struct {
	Path         string   `json:"path"`
	Dependencies []string `json:"dependencies"`
}

func DecodeStrict[T any](b []byte, label string) (T, error) {
	var zero T
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&zero); err != nil {
		return zero, fmt.Errorf("%s: %w", label, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return zero, fmt.Errorf("%s: unexpected extra JSON content", label)
		}
		return zero, fmt.Errorf("%s: %w", label, err)
	}
	return zero, nil
}

