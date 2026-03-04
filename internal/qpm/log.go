package qpm

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type Logger interface {
	Infof(format string, args ...any)
	Verbosef(format string, args ...any)
	VerboseEnabled() bool
}

type StdLogger struct {
	verbose bool
	out     io.Writer
	mu      sync.Mutex
}

func NewStdLogger(verbose bool) *StdLogger {
	return &StdLogger{
		verbose: verbose,
		out:     os.Stderr,
	}
}

func (l *StdLogger) VerboseEnabled() bool { return l != nil && l.verbose }

func (l *StdLogger) Infof(format string, args ...any) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, format+"\n", args...)
}

func (l *StdLogger) Verbosef(format string, args ...any) {
	if l == nil || !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, format+"\n", args...)
}

type NopLogger struct{}

func (NopLogger) Infof(string, ...any)         {}
func (NopLogger) Verbosef(string, ...any)      {}
func (NopLogger) VerboseEnabled() bool         { return false }

