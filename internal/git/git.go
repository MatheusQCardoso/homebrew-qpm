package git

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	Verbose bool
}

func (r Runner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	if r.Verbose {
		pretty := redactArgs(args)
		cwd := dir
		if cwd == "" {
			cwd = "."
		}
		fmt.Fprintf(os.Stderr, "[git] (%s) git %s\n", cwd, strings.Join(pretty, " "))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg != "" {
			return stdout.String(), fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
		}
		return stdout.String(), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return stdout.String(), nil
}

func redactArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, redactArg(a))
	}
	return out
}

func redactArg(a string) string {
	u, err := url.Parse(a)
	if err != nil || u == nil || u.Scheme == "" {
		return a
	}
	if u.User == nil {
		return a
	}
	if _, ok := u.User.Password(); !ok {
		return a
	}
	u.User = url.UserPassword(u.User.Username(), "***")
	return u.String()
}

type SparseCloneOptions struct {
	Repo string
	Ref  string // branch, tag, or revision (commit)

	// SparsePaths must use repo-relative paths ("Package.swift", "Sources", ...).
	SparsePaths []string
}

// SparseClone performs a minimal clone and materializes only sparse paths.
// It always checks out the provided ref (if non-empty).
func SparseClone(ctx context.Context, r Runner, dstDir string, opt SparseCloneOptions) error {
	if opt.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	if len(opt.SparsePaths) == 0 {
		return fmt.Errorf("SparsePaths is required")
	}

	cloneArgs := []string{
		"clone",
		"--filter=blob:none",
		"--no-checkout",
		"--sparse",
		opt.Repo,
		dstDir,
	}
	if _, err := r.Run(ctx, "", cloneArgs...); err != nil {
		return err
	}

	// Configure sparse paths before checkout to avoid populating unwanted files.
	args := []string{"sparse-checkout", "set", "--no-cone", "--"}
	args = append(args, opt.SparsePaths...)
	if _, err := r.Run(ctx, dstDir, args...); err != nil {
		return err
	}

	if opt.Ref != "" {
		// Allow tags/branches; for commit SHA, checkout works too.
		if _, err := r.Run(ctx, dstDir, "checkout", opt.Ref); err != nil {
			return err
		}
	} else {
		if _, err := r.Run(ctx, dstDir, "checkout"); err != nil {
			return err
		}
	}

	return nil
}

func EnsureSparsePaths(ctx context.Context, r Runner, repoDir string, ref string, sparsePaths []string) error {
	if len(sparsePaths) == 0 {
		return nil
	}
	args := []string{"sparse-checkout", "set", "--no-cone", "--"}
	args = append(args, sparsePaths...)
	if _, err := r.Run(ctx, repoDir, args...); err != nil {
		return err
	}
	if ref != "" {
		if _, err := r.Run(ctx, repoDir, "checkout", ref); err != nil {
			return err
		}
	} else {
		if _, err := r.Run(ctx, repoDir, "checkout"); err != nil {
			return err
		}
	}
	return nil
}

