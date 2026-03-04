package swiftpm

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/model"
)

func RenderPackageSwiftFromQPM(m model.QpmPackageManifest) ([]byte, error) {
	if strings.TrimSpace(m.SwiftToolsVersion) == "" {
		return nil, fmt.Errorf("missing swift-tools-version")
	}
	if m.Package.Name == "" {
		return nil, fmt.Errorf("missing package.name")
	}

	var out bytes.Buffer

	out.WriteString("// swift-tools-version: ")
	out.WriteString(strings.TrimSpace(m.SwiftToolsVersion))
	out.WriteString("\n")
	out.WriteString("import PackageDescription\n")

	if strings.TrimSpace(m.Version) != "" {
		out.WriteString("let version = ")
		out.WriteString(strconv.Quote(strings.TrimSpace(m.Version)))
		out.WriteString("\n")
	}

	out.WriteString("let package = Package(\n")
	out.WriteString("    name: ")
	out.WriteString(strconv.Quote(m.Package.Name))
	out.WriteString(",\n")

	iosMajor := parseIOSMajor(m.Package.MinIOSVersion)
	out.WriteString("    platforms: [\n")
	out.WriteString("        .iOS(.v")
	out.WriteString(strconv.Itoa(iosMajor))
	out.WriteString(")\n")
	out.WriteString("    ],\n")

	out.WriteString("    products: [\n")
	for i, p := range m.Package.Products {
		if strings.ToLower(p.Type) != "library" {
			return nil, fmt.Errorf("unsupported product type %q", p.Type)
		}
		out.WriteString("        .library(\n")
		out.WriteString("            name: ")
		out.WriteString(strconv.Quote(p.Name))
		out.WriteString(",\n")
		out.WriteString("            targets: [")
		for j, t := range p.Targets {
			if j > 0 {
				out.WriteString(", ")
			}
			out.WriteString(strconv.Quote(t))
		}
		out.WriteString("]\n")
		out.WriteString("        )")
		if i < len(m.Package.Products)-1 {
			out.WriteString(",")
		}
		out.WriteString("\n")
	}
	out.WriteString("    ],\n")

	depNames := make([]string, 0, len(m.Package.Dependencies))
	for name := range m.Package.Dependencies {
		depNames = append(depNames, name)
	}
	sort.Strings(depNames)

	out.WriteString("    dependencies: [\n")
	for i, dep := range depNames {
		out.WriteString("        .package(path: ")
		out.WriteString(strconv.Quote("../" + dep))
		out.WriteString(")")
		if i < len(depNames)-1 {
			out.WriteString(",")
		}
		out.WriteString("\n")
	}
	out.WriteString("    ],\n")

	localTargetNames := map[string]bool{}
	for name := range m.Package.Targets {
		localTargetNames[name] = true
	}
	for name := range m.Package.TestTargets {
		localTargetNames[name] = true
	}

	out.WriteString("    targets: [\n")

	targetNames := make([]string, 0, len(m.Package.Targets))
	for name := range m.Package.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	for _, name := range targetNames {
		t := m.Package.Targets[name]
		out.WriteString("        .target(\n")
		out.WriteString("            name: ")
		out.WriteString(strconv.Quote(name))
		out.WriteString(",\n")
		out.WriteString("            dependencies: [\n")
		for j, dep := range t.Dependencies {
			out.WriteString("                ")
			out.WriteString(renderTargetDependency(dep, localTargetNames))
			if j < len(t.Dependencies)-1 {
				out.WriteString(",")
			}
			out.WriteString("\n")
		}
		out.WriteString("            ],\n")
		out.WriteString("            path: ")
		out.WriteString(strconv.Quote(t.Path))
		out.WriteString("\n")
		out.WriteString("        ),\n")
	}

	testNames := make([]string, 0, len(m.Package.TestTargets))
	for name := range m.Package.TestTargets {
		testNames = append(testNames, name)
	}
	sort.Strings(testNames)

	for idx, name := range testNames {
		t := m.Package.TestTargets[name]
		out.WriteString("        .testTarget(\n")
		out.WriteString("            name: ")
		out.WriteString(strconv.Quote(name))
		out.WriteString(",\n")
		out.WriteString("            dependencies: [\n")
		for j, dep := range t.Dependencies {
			out.WriteString("                ")
			out.WriteString(renderTargetDependency(dep, localTargetNames))
			if j < len(t.Dependencies)-1 {
				out.WriteString(",")
			}
			out.WriteString("\n")
		}
		out.WriteString("            ],\n")
		out.WriteString("            path: ")
		out.WriteString(strconv.Quote(t.Path))
		out.WriteString("\n")
		out.WriteString("        )")
		if idx < len(testNames)-1 {
			out.WriteString(",")
		}
		out.WriteString("\n")
	}

	out.WriteString("    ]\n")
	out.WriteString(")\n")

	return out.Bytes(), nil
}

func renderTargetDependency(dep string, localTargets map[string]bool) string {
	if localTargets[dep] {
		return fmt.Sprintf(".target(name: %s)", strconv.Quote(dep))
	}
	return fmt.Sprintf(".product(name: %s, package: %s)", strconv.Quote(dep), strconv.Quote(dep))
}

func parseIOSMajor(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 15
	}
	parts := strings.Split(v, ".")
	major, err := strconv.Atoi(parts[0])
	if err != nil || major <= 0 {
		return 15
	}
	return major
}

