package rivet

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBlockLifecycleAndOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	project := t.TempDir()
	s := Store{Project: project}
	source := t.TempDir()
	script := filepath.Join(source, "run.sh")
	check := filepath.Join(source, "verify.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nset -eu\nprintf '%s\\n' \"$1\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(check, []byte("#!/bin/sh\nset -eu\ntest \"$(sh \"$RIVET_SCRIPT\" hello)\" = hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("echo", "Echo the argument", script, check); err != nil {
		t.Fatal(err)
	}
	ignore, err := os.ReadFile(filepath.Join(project, ".rivet", ".gitignore"))
	if err != nil || !strings.Contains(string(ignore), "logs/") {
		t.Fatalf("gitignore: %q, %v", ignore, err)
	}
	if _, err := s.BlockCommand(context.Background(), "echo", nil); err == nil {
		t.Fatal("unverified block ran")
	}
	if err := s.Verify(context.Background(), "echo", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	status, err := s.Show("echo")
	if err != nil || !status.Verified {
		t.Fatalf("status: %+v, %v", status, err)
	}
	argument := "$(touch should-not-exist); hello"
	cmd, err := s.BlockCommand(context.Background(), "echo", []string{argument})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Execute(context.Background(), cmd, 4)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.Summary) != argument {
		t.Fatalf("summary: %q", result.Summary)
	}
	if _, err := os.Stat(filepath.Join(project, "should-not-exist")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("argument executed as shell")
	}
	if _, err := os.Stat(result.LogPath); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(project, ".rivet", "blocks", "echo", "run.sh")
	if err := os.WriteFile(installed, []byte("#!/bin/sh\necho changed\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BlockCommand(context.Background(), "echo", nil); err == nil {
		t.Fatal("changed script accepted")
	}
	if err := s.Verify(context.Background(), "echo", &bytes.Buffer{}); err == nil {
		t.Fatal("bad recheck passed")
	}
	if _, err := s.BlockCommand(context.Background(), "echo", nil); err == nil {
		t.Fatal("failed recheck left receipt")
	}
}

func TestExecuteFailureKeepsFullLogAndExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	s := Store{Project: t.TempDir()}
	cmd := exec.Command("sh", "-c", "printf 'first\\nsecond\\nthird\\nfourth\\nfifth\\n'; exit 23")
	result, err := s.Execute(context.Background(), cmd, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 23 || !strings.Contains(result.Summary, "lines omitted") {
		t.Fatalf("result: %+v", result)
	}
	raw, err := os.ReadFile(result.LogPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "third") || !strings.Contains(string(raw), "fifth") {
		t.Fatalf("raw log: %q", raw)
	}
}

func TestSummarizeGoTest(t *testing.T) {
	input := "noise 1\nnoise 2\nnoise 3\nnoise 4\nok  \tmodule/pkg  1s\n"
	summary, lines, err := Summarize(strings.NewReader(input), []string{"go", "test", "./..."}, 0, 2)
	if err != nil || lines != 5 || !strings.Contains(summary, "1 packages passed") {
		t.Fatalf("summary: %q, %d, %v", summary, lines, err)
	}
	failure, _, err := Summarize(strings.NewReader(input), []string{"go", "test", "./..."}, 1, 2)
	if err != nil || strings.Contains(failure, "packages passed") {
		t.Fatalf("failure summary: %q, %v", failure, err)
	}
}

func TestAddRejectsInvalidSources(t *testing.T) {
	s := Store{Project: t.TempDir()}
	source := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(source, []byte("exit 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("../escape", "bad", source, source); err == nil {
		t.Fatal("traversal accepted")
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(t.TempDir(), "link.sh")
		if err := os.Symlink(source, link); err != nil {
			t.Fatal(err)
		}
		if err := s.Add("safe", "bad", link, source); err == nil {
			t.Fatal("symlink source accepted")
		}
	}
}
