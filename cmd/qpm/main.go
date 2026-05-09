package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MatheusQCardoso/homebrew-qpm/internal/qpm"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	ctx := context.Background()
	os.Exit(run(ctx, os.Args[1:]))
}

func run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		usage(os.Stdout)
		return 0
	}

	switch args[0] {
	case "install":
		return runInstall(ctx, args[1:])
	case "graph":
		return runGraph(ctx, args[1:])
	case "clean":
		return runClean(ctx, args[1:])
	case "version", "--version":
		return runVersion(os.Stdout)
	case "-h", "--help", "help":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		usage(os.Stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `NAME
  qpm - Quirino's Package Manager

SYNOPSIS
  qpm <command> [options]

COMMANDS
  install   Resolve deps and materialize QPackages/
  graph     Print the resolved dependency graph (JSON)
  clean     Delete the packages directory and qpm-generated leftovers
  version   Print qpm version and build metadata
  help      Show this help

OPTIONS
  Common:
    --file <path>          Path to Quirino.json (default: Quirino.json)

  install:
    --verbose              Verbose output

EXAMPLES
  qpm install
  qpm install --file ./Quirino.json --verbose
  qpm graph
  qpm version
  qpm clean
`)
}

func runInstall(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	quirinoFile := fs.String("file", "Quirino.json", "path to Quirino.json")
	verbose := fs.Bool("verbose", false, "verbose output")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	absFile, err := filepath.Abs(*quirinoFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	opts := qpm.InstallOptions{
		QuirinoJSONPath: absFile,
		Verbose:         *verbose,
	}
	if err := qpm.Install(ctx, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runGraph(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	quirinoFile := fs.String("file", "Quirino.json", "path to Quirino.json")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	absFile, err := filepath.Abs(*quirinoFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	out, err := qpm.BuildGraphJSON(ctx, qpm.BuildGraphOptions{
		QuirinoJSONPath: absFile,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Println(string(out))
	return 0
}

func runClean(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	quirinoFile := fs.String("file", "Quirino.json", "path to Quirino.json")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	absFile, err := filepath.Abs(*quirinoFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := qpm.Clean(ctx, qpm.CleanOptions{
		QuirinoJSONPath: absFile,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runVersion(w io.Writer) int {
	fmt.Fprintf(w, "qpm %s\ncommit: %s\nbuilt: %s\n", version, commit, buildTime)
	return 0
}
