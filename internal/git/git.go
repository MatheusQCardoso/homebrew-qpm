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

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		errorMessage := strings.TrimSpace(stderrBuffer.String())
		if errorMessage == "" {
			errorMessage = strings.TrimSpace(stdoutBuffer.String())
		}
		if errorMessage != "" {
			return stdoutBuffer.String(), fmt.Errorf("git %s: %s", strings.Join(args, " "), errorMessage)
		}
		return stdoutBuffer.String(), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return stdoutBuffer.String(), nil
}

func redactArgs(args []string) []string {
	redactedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		redactedArgs = append(redactedArgs, redactArg(arg))
	}
	return redactedArgs
}

func redactArg(arg string) string {
	parsedURL, err := url.Parse(arg)
	if err != nil || parsedURL == nil || parsedURL.Scheme == "" {
		return arg
	}
	if parsedURL.User == nil {
		return arg
	}
	if _, ok := parsedURL.User.Password(); !ok {
		return arg
	}
	parsedURL.User = url.UserPassword(parsedURL.User.Username(), "***")
	return parsedURL.String()
}

type SparseCloneOptions struct {
	Repo        string
	Ref         string
	SparsePaths []string
}

func SparseClone(ctx context.Context, r Runner, dstDir string, opt SparseCloneOptions) error {
	if opt.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	if len(opt.SparsePaths) == 0 {
		return fmt.Errorf("SparsePaths is required")
	}

	cloneArguments := []string{
		"clone",
		"--filter=blob:none",
		"--no-checkout",
		"--sparse",
		opt.Repo,
		dstDir,
	}
	if _, err := r.Run(ctx, "", cloneArguments...); err != nil {
		return err
	}

	sparseCheckoutArgs := []string{"sparse-checkout", "set", "--no-cone", "--"}
	sparseCheckoutArgs = append(sparseCheckoutArgs, opt.SparsePaths...)
	if _, err := r.Run(ctx, dstDir, sparseCheckoutArgs...); err != nil {
		return err
	}

	if opt.Ref != "" {
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
	sparseCheckoutArgs := []string{"sparse-checkout", "set", "--no-cone", "--"}
	sparseCheckoutArgs = append(sparseCheckoutArgs, sparsePaths...)
	if _, err := r.Run(ctx, repoDir, sparseCheckoutArgs...); err != nil {
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
