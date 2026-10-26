package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GrayCodeAI/rivet/internal/rivet"
)

const usage = `Rivet: reusable checked scripts and compact command output

Usage:
  rivet init [--project DIR]
  rivet add [--project DIR] --description TEXT --script FILE --check FILE NAME
  rivet verify [--project DIR] [--timeout DURATION] NAME
  rivet run [--project DIR] [--max-lines N] NAME [ARG...]
  rivet exec [--project DIR] [--max-lines N] -- COMMAND [ARG...]
  rivet list [--project DIR]
  rivet show [--project DIR] NAME

The complete output of run and exec is saved under .rivet/logs/.
`

func main() { os.Exit(run(os.Args[1:])) }

func run(argv []string) int {
	if len(argv) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	command := argv[0]
	if command == "help" || command == "--help" {
		fmt.Print(usage)
		return 0
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	project := fs.String("project", ".", "project directory")
	var description, script, check *string
	var timeout *time.Duration
	var maxLines *int
	switch command {
	case "add":
		description = fs.String("description", "", "block description")
		script = fs.String("script", "", "script to import")
		check = fs.String("check", "", "verification script to import")
	case "verify":
		timeout = fs.Duration("timeout", 2*time.Minute, "verification timeout")
	case "run", "exec":
		maxLines = fs.Int("max-lines", 24, "maximum visible output lines")
	case "init", "list", "show":
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", command, usage)
		return 2
	}
	if err := fs.Parse(argv[1:]); err != nil {
		return 2
	}
	args := fs.Args()
	s := rivet.Store{Project: *project}
	var err error
	switch command {
	case "init":
		if len(args) != 0 {
			return badArgs()
		}
		err = s.Init()
		if err == nil {
			fmt.Println("Initialized .rivet")
		}
	case "add":
		if len(args) != 1 || *script == "" || *check == "" {
			return badArgs()
		}
		err = s.Add(args[0], *description, *script, *check)
		if err == nil {
			fmt.Printf("Added %s; run rivet verify %s\n", args[0], args[0])
		}
	case "verify":
		if len(args) != 1 || *timeout <= 0 {
			return badArgs()
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		err = s.Verify(ctx, args[0], os.Stderr)
		if err == nil {
			fmt.Printf("Verified %s\n", args[0])
		}
	case "run":
		if len(args) < 1 {
			return badArgs()
		}
		var cmd *exec.Cmd
		cmd, err = s.BlockCommand(context.Background(), args[0], args[1:])
		if err == nil {
			return execute(s, cmd, *maxLines)
		}
	case "exec":
		if len(args) < 1 {
			return badArgs()
		}
		root, rootErr := s.Root()
		if rootErr != nil {
			err = rootErr
			break
		}
		cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
		cmd.Dir = root
		return execute(s, cmd, *maxLines)
	case "list":
		if len(args) != 0 {
			return badArgs()
		}
		var statuses []rivet.Status
		statuses, err = s.List()
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(statuses)
		}
	case "show":
		if len(args) != 1 {
			return badArgs()
		}
		var status rivet.Status
		status, err = s.Show(args[0])
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(status)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "rivet:", err)
		return 1
	}
	return 0
}

func execute(s rivet.Store, cmd *exec.Cmd, maxLines int) int {
	result, err := s.Execute(context.Background(), cmd, maxLines)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rivet:", err)
		return 1
	}
	if strings.TrimSpace(result.Summary) != "" {
		fmt.Println(result.Summary)
	}
	fmt.Fprintf(os.Stderr, "rivet: exit=%d lines=%d raw=%s\n", result.ExitCode, result.Lines, result.LogPath)
	return result.ExitCode
}

func badArgs() int { fmt.Fprint(os.Stderr, usage); return 2 }
