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
		if ctx.Err() != nil {
			command := strings.Join(args, " ")
			return stdoutBuffer.String(), fmt.Errorf("git %s: %w", command, ctx.Err())
		}
		errorMessage := strings.TrimSpace(stderrBuffer.String())
		if errorMessage == "" {
			errorMessage = strings.TrimSpace(stdoutBuffer.String())
		}
		command := strings.Join(args, " ")
		baseErr := fmt.Errorf("git %s: %s", command, errorMessage)
		if hints := gitCommandHints(args, errorMessage); hints != "" {
			return stdoutBuffer.String(), fmt.Errorf("%w\n\n%s", baseErr, hints)
		}
		return stdoutBuffer.String(), baseErr
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

func gitCommandHints(args []string, stderr string) string {
	if len(args) == 0 {
		return ""
	}

	switch args[0] {
	case "clone":
		return cloneHints(args, stderr)
	default:
		return ""
	}
}

func cloneHints(args []string, stderr string) string {
	repo := cloneRepoArg(args)
	if repo == "" {
		repo = "<repository>"
	}

	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "authentication failed") || strings.Contains(lower, "could not read from remote repository") {
		return fmt.Sprintf("❌ Failed to clone %s\n  • Check your SSH keys: ssh-add -K ~/.ssh/id_rsa\n  • Or use HTTPS: modify Quirino.json to use https:// URL\n  • Confirm your Git credentials and host access.", repo)
	}
	if strings.Contains(lower, "repository not found") || strings.Contains(lower, "remote repository not found") {
		return fmt.Sprintf("❌ Failed to clone %s\n  • Verify the repository URL and access rights\n  • Ensure the repository exists and your account has read permission.", repo)
	}
	if strings.Contains(lower, "could not resolve host") || strings.Contains(lower, "unable to access") {
		return fmt.Sprintf("❌ Failed to clone %s\n  • Check network connectivity and DNS resolution\n  • If you are behind a proxy, verify Git proxy settings.", repo)
	}
	if strings.Contains(lower, "connection timed out") || strings.Contains(lower, "timed out") {
		return fmt.Sprintf("❌ Failed to clone %s\n  • The remote host did not respond in time\n  • Check your network connection or try again later.", repo)
	}
	return fmt.Sprintf("❌ Failed to clone %s\n  • Verify your Git URL and network connectivity\n  • Check repository access or try HTTPS if SSH is blocked.", repo)
}

func cloneRepoArg(args []string) string {
	for i := 1; i < len(args)-1; i++ {
		if !strings.HasPrefix(args[i], "-") {
			return args[i]
		}
	}
	return ""
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
