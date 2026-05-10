package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
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
	normalizedSpec := s
	normalizedSpec.Type = DependencyType(strings.ToLower(string(normalizedSpec.Type)))
	normalizedSpec.Repo = strings.TrimSpace(normalizedSpec.Repo)
	path := strings.TrimSpace(normalizedSpec.Path)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~") {
		path = strings.TrimRight(path, "/")
	} else if !isRelativePath(path) {
		path = strings.Trim(path, "/")
	}
	normalizedSpec.Path = path
	normalizedSpec.Branch = strings.TrimSpace(normalizedSpec.Branch)
	normalizedSpec.Tag = strings.TrimSpace(normalizedSpec.Tag)
	normalizedSpec.Revision = strings.TrimSpace(normalizedSpec.Revision)
	if normalizedSpec.Type == "" {
		normalizedSpec.Type = DependencyTypeQPM
	}
	return normalizedSpec
}

func isRelativePath(p string) bool {
	if p == "" {
		return false
	}
	return !strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "~")
}

func (s DepSpec) Validate() error {
	spec := s.Normalized()
	if err := spec.Ref.Validate(); err != nil {
		return err
	}
	switch spec.Type {
	case DependencyTypeQPM, DependencyTypeSPM:
	default:
		return fmt.Errorf("unsupported type %q (supported: %q, %q)", spec.Type, DependencyTypeQPM, DependencyTypeSPM)
	}

	if spec.Repo != "" {
		return nil
	}

	if spec.Path == "" {
		return fmt.Errorf("either repo (remote) or path (local) is required")
	}
	if spec.Branch != "" || spec.Tag != "" || spec.Revision != "" {
		return fmt.Errorf("local dependency cannot specify branch/tag/revision")
	}
	if !strings.HasPrefix(spec.Path, "/") && !strings.HasPrefix(spec.Path, "~") && !isRelativePath(spec.Path) {
		return fmt.Errorf("local dependency path must be absolute (or start with ~) or relative: %q", spec.Path)
	}
	return nil
}

type QuirinoManifest struct {
	PackagesDir           string             `json:"packagesDir,omitempty"`
	NetworkTimeoutSeconds SecondsDuration    `json:"networkTimeoutSeconds,omitempty"`
	Dependencies          map[string]DepSpec `json:"dependencies"`
}

func (m QuirinoManifest) NetworkTimeout() time.Duration {
	if m.NetworkTimeoutSeconds.Valid {
		return m.NetworkTimeoutSeconds.Duration
	}
	return 30 * time.Second
}

type SecondsDuration struct {
	Duration time.Duration
	Valid    bool
}

func (d *SecondsDuration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var numeric float64
	if err := json.Unmarshal(data, &numeric); err == nil {
		d.Duration = time.Duration(numeric * float64(time.Second))
		d.Valid = true
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if str == "" {
		return nil
	}

	numeric, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("invalid networkTimeoutSeconds: %w", err)
	}
	d.Duration = time.Duration(numeric * float64(time.Second))
	d.Valid = true
	return nil
}

type QpmPackageManifest struct {
	SwiftToolsVersion string                `json:"swift-tools-version"`
	Version           string                `json:"version"`
	Package           QpmPackageDeclaration `json:"package"`
}

type QpmPackageDeclaration struct {
	Name          string                   `json:"name"`
	MinIOSVersion string                   `json:"min-ios-version"`
	Products      []QpmProduct             `json:"products"`
	Dependencies  map[string]DepSpec       `json:"dependencies"`
	Targets       map[string]QpmTarget     `json:"targets"`
	TestTargets   map[string]QpmTestTarget `json:"testTargets"`
}

type QpmProduct struct {
	Type    string   `json:"type"`
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
	var result T
	decoder := json.NewDecoder(bytes.NewReader(b))
	if err := decoder.Decode(&result); err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}
	var extraContent any
	if err := decoder.Decode(&extraContent); err != io.EOF {
		if err == nil {
			return result, fmt.Errorf("%s: unexpected extra JSON content", label)
		}
		return result, fmt.Errorf("%s: %w", label, err)
	}
	return result, nil
}
