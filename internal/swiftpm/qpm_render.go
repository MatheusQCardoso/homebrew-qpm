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

	var buffer bytes.Buffer

	buffer.WriteString("// swift-tools-version: ")
	buffer.WriteString(strings.TrimSpace(m.SwiftToolsVersion))
	buffer.WriteString("\n")
	buffer.WriteString("import PackageDescription\n")

	if strings.TrimSpace(m.Version) != "" {
		buffer.WriteString("let version = ")
		buffer.WriteString(strconv.Quote(strings.TrimSpace(m.Version)))
		buffer.WriteString("\n")
	}

	buffer.WriteString("let package = Package(\n")
	buffer.WriteString("    name: ")
	buffer.WriteString(strconv.Quote(m.Package.Name))
	buffer.WriteString(",\n")

	iosMajorVersion := parseIOSMajor(m.Package.MinIOSVersion)
	buffer.WriteString("    platforms: [\n")
	buffer.WriteString("        .iOS(.v")
	buffer.WriteString(strconv.Itoa(iosMajorVersion))
	buffer.WriteString(")\n")
	buffer.WriteString("    ],\n")

	buffer.WriteString("    products: [\n")
	for index, product := range m.Package.Products {
		if strings.ToLower(product.Type) != "library" {
			return nil, fmt.Errorf("unsupported product type %q", product.Type)
		}
		buffer.WriteString("        .library(\n")
		buffer.WriteString("            name: ")
		buffer.WriteString(strconv.Quote(product.Name))
		buffer.WriteString(",\n")
		buffer.WriteString("            targets: [")
		for targetIndex, target := range product.Targets {
			if targetIndex > 0 {
				buffer.WriteString(", ")
			}
			buffer.WriteString(strconv.Quote(target))
		}
		buffer.WriteString("]\n")
		buffer.WriteString("        )")
		if index < len(m.Package.Products)-1 {
			buffer.WriteString(",")
		}
		buffer.WriteString("\n")
	}
	buffer.WriteString("    ],\n")

	dependencyNames := make([]string, 0, len(m.Package.Dependencies))
	for dependencyName := range m.Package.Dependencies {
		dependencyNames = append(dependencyNames, dependencyName)
	}
	sort.Strings(dependencyNames)

	buffer.WriteString("    dependencies: [\n")
	for index, dependency := range dependencyNames {
		buffer.WriteString("        .package(path: ")
		buffer.WriteString(strconv.Quote("../" + dependency))
		buffer.WriteString(")")
		if index < len(dependencyNames)-1 {
			buffer.WriteString(",")
		}
		buffer.WriteString("\n")
	}
	buffer.WriteString("    ],\n")

	localTargetNames := map[string]bool{}
	for targetName := range m.Package.Targets {
		localTargetNames[targetName] = true
	}
	for testTargetName := range m.Package.TestTargets {
		localTargetNames[testTargetName] = true
	}

	buffer.WriteString("    targets: [\n")

	targetNames := make([]string, 0, len(m.Package.Targets))
	for targetName := range m.Package.Targets {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)

	for _, targetName := range targetNames {
		target := m.Package.Targets[targetName]
		buffer.WriteString("        .target(\n")
		buffer.WriteString("            name: ")
		buffer.WriteString(strconv.Quote(targetName))
		buffer.WriteString(",\n")
		buffer.WriteString("            dependencies: [\n")
		for dependencyIndex, dependency := range target.Dependencies {
			buffer.WriteString("                ")
			buffer.WriteString(renderTargetDependency(dependency, localTargetNames))
			if dependencyIndex < len(target.Dependencies)-1 {
				buffer.WriteString(",")
			}
			buffer.WriteString("\n")
		}
		buffer.WriteString("            ],\n")
		buffer.WriteString("            path: ")
		buffer.WriteString(strconv.Quote(target.Path))
		buffer.WriteString("\n")
		buffer.WriteString("        ),\n")
	}

	testTargetNames := make([]string, 0, len(m.Package.TestTargets))
	for testTargetName := range m.Package.TestTargets {
		testTargetNames = append(testTargetNames, testTargetName)
	}
	sort.Strings(testTargetNames)

	for index, testTargetName := range testTargetNames {
		testTarget := m.Package.TestTargets[testTargetName]
		buffer.WriteString("        .testTarget(\n")
		buffer.WriteString("            name: ")
		buffer.WriteString(strconv.Quote(testTargetName))
		buffer.WriteString(",\n")
		buffer.WriteString("            dependencies: [\n")
		for dependencyIndex, dependency := range testTarget.Dependencies {
			buffer.WriteString("                ")
			buffer.WriteString(renderTargetDependency(dependency, localTargetNames))
			if dependencyIndex < len(testTarget.Dependencies)-1 {
				buffer.WriteString(",")
			}
			buffer.WriteString("\n")
		}
		buffer.WriteString("            ],\n")
		buffer.WriteString("            path: ")
		buffer.WriteString(strconv.Quote(testTarget.Path))
		buffer.WriteString("\n")
		buffer.WriteString("        )")
		if index < len(testTargetNames)-1 {
			buffer.WriteString(",")
		}
		buffer.WriteString("\n")
	}

	buffer.WriteString("    ]\n")
	buffer.WriteString(")\n")

	return buffer.Bytes(), nil
}

func renderTargetDependency(dependency string, localTargets map[string]bool) string {
	if localTargets[dependency] {
		return fmt.Sprintf(".target(name: %s)", strconv.Quote(dependency))
	}
	return fmt.Sprintf(".product(name: %s, package: %s)", strconv.Quote(dependency), strconv.Quote(dependency))
}

func parseIOSMajor(version string) int {
	version = strings.TrimSpace(version)
	if version == "" {
		return 15
	}
	parts := strings.Split(version, ".")
	major, err := strconv.Atoi(parts[0])
	if err != nil || major <= 0 {
		return 15
	}
	return major
}
