package rivet

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Result struct {
	LogPath  string `json:"log_path"`
	ExitCode int    `json:"exit_code"`
	RawBytes int64  `json:"raw_bytes"`
	Lines    int    `json:"lines"`
	Summary  string `json:"summary"`
}

// Execute keeps complete output in a local log and returns a bounded view.
func (s Store) Execute(ctx context.Context, cmd *exec.Cmd, maxLines int) (Result, error) {
	if maxLines < 2 {
		return Result{}, errors.New("max-lines must be at least 2")
	}
	logDir := s.LogDir
	if logDir == "" {
		if err := s.Init(); err != nil {
			return Result{}, err
		}
		base, err := s.Base()
		if err != nil {
			return Result{}, err
		}
		logDir = filepath.Join(base, "logs")
	} else if err := os.MkdirAll(logDir, 0700); err != nil {
		return Result{}, err
	}
	log, err := os.CreateTemp(logDir, "run-*.log")
	if err != nil {
		return Result{}, err
	}
	path := log.Name()
	if err := log.Chmod(0600); err != nil {
		_ = log.Close()
		return Result{}, err
	}
	cmd.Stdout, cmd.Stderr = log, log
	runErr := cmd.Run()
	if err := log.Close(); err != nil {
		return Result{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, err
	}
	exitCode := 0
	if runErr != nil {
		var exit *exec.ExitError
		if errors.As(runErr, &exit) {
			exitCode = exit.ExitCode()
		} else if ctx.Err() != nil {
			return Result{}, ctx.Err()
		} else {
			return Result{}, runErr
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()
	summary, lines, err := Summarize(file, cmd.Args, exitCode, maxLines)
	if err != nil {
		return Result{}, err
	}
	return Result{LogPath: path, ExitCode: exitCode, RawBytes: info.Size(), Lines: lines, Summary: summary}, nil
}

// Summarize preserves all output for short commands and the first and last
// lines for long commands. A successful go test run gets a package summary.
func Summarize(r io.Reader, command []string, exitCode, maxLines int) (string, int, error) {
	if maxLines < 2 {
		return "", 0, errors.New("max-lines must be at least 2")
	}
	reader := bufio.NewReader(r)
	headCount := maxLines / 2
	tailCount := maxLines - headCount
	var head, tail []string
	lines, packages := 0, 0
	goTest := len(command) >= 2 && filepath.Base(command[0]) == "go" && command[1] == "test"
	for {
		line, err := boundedLine(reader, 500)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, err
		}
		lines++
		if goTest && (strings.HasPrefix(line, "ok  ") || strings.HasPrefix(line, "?   ")) {
			packages++
		}
		if len(head) < headCount {
			head = append(head, line)
		} else {
			if len(tail) == tailCount {
				tail = tail[1:]
			}
			tail = append(tail, line)
		}
	}
	if goTest && exitCode == 0 && packages > 0 && lines > maxLines {
		return fmt.Sprintf("go test: %d packages passed; %d output lines (full log retained)", packages, lines), lines, nil
	}
	if lines <= maxLines {
		return strings.Join(append(head, tail...), "\n"), lines, nil
	}
	omitted := lines - len(head) - len(tail)
	all := append(head, fmt.Sprintf("... %d lines omitted (full log retained) ...", omitted))
	all = append(all, tail...)
	return strings.Join(all, "\n"), lines, nil
}

func boundedLine(r *bufio.Reader, max int) (string, error) {
	var out []byte
	seen := false
	truncated := false
	for {
		chunk, more, err := r.ReadLine()
		if err == io.EOF && !seen {
			return "", io.EOF
		}
		if err != nil && err != io.EOF {
			return "", err
		}
		seen = true
		space := max - len(out)
		if len(chunk) > space {
			out = append(out, chunk[:space]...)
			truncated = true
		} else {
			out = append(out, chunk...)
		}
		if !more {
			if truncated {
				return string(out) + "…", nil
			}
			return string(out), nil
		}
	}
}
