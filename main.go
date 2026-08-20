package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"

	"agres/internal/agent"
	"agres/internal/session"
	"agres/internal/tui"
)

var version = "0.4.0"

type cliOptions struct {
	limit       int
	allProjects bool
	showVersion bool
	showHelp    bool
}

func parseArgs(args []string) (cliOptions, error) {
	opts := cliOptions{limit: 10}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-v" || arg == "--version" || arg == "-version":
			opts.showVersion = true
			return opts, nil
		case arg == "-h" || arg == "--help" || arg == "-help":
			opts.showHelp = true
			return opts, nil
		case arg == "-a" || arg == "--all":
			opts.allProjects = true
		case arg == "-n" || arg == "-l" || arg == "--limit" || arg == "-limit" || arg == "--number" || arg == "-number":
			if i+1 >= len(args) {
				return cliOptions{}, fmt.Errorf("flag '%s' requires an integer argument", arg)
			}
			i++
			val, convErr := strconv.Atoi(args[i])
			if convErr != nil || val <= 0 {
				return cliOptions{}, fmt.Errorf("invalid count '%s': must be a positive integer", args[i])
			}
			opts.limit = val
		case strings.HasPrefix(arg, "-n=") || strings.HasPrefix(arg, "-l=") || strings.HasPrefix(arg, "--limit=") || strings.HasPrefix(arg, "-limit=") || strings.HasPrefix(arg, "--number=") || strings.HasPrefix(arg, "-number="):
			parts := strings.SplitN(arg, "=", 2)
			val, convErr := strconv.Atoi(parts[1])
			if convErr != nil || val <= 0 {
				return cliOptions{}, fmt.Errorf("invalid count '%s': must be a positive integer", parts[1])
			}
			opts.limit = val
		case !strings.HasPrefix(arg, "-"):
			val, convErr := strconv.Atoi(arg)
			if convErr == nil {
				if val <= 0 {
					return cliOptions{}, fmt.Errorf("invalid count '%s': must be a positive integer", arg)
				}
				opts.limit = val
			} else {
				return cliOptions{}, fmt.Errorf("unknown argument '%s'", arg)
			}
		default:
			return cliOptions{}, fmt.Errorf("unknown flag '%s'", arg)
		}
	}

	return opts, nil
}

func printHelp() {
	fmt.Printf("agres %s\n", version)
	fmt.Println("Resume CLI coding agent sessions from the current directory or all projects.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agres [options] [count]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -a, --all                 Show sessions from all projects")
	fmt.Println("  -n, -l, --limit <count>   Number of history items to show (default: 10)")
	fmt.Println("  -v, --version             Show version information")
	fmt.Println("  -h, --help                Show help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agres")
	fmt.Println("  agres --all")
	fmt.Println("  agres -a --limit 20")
	fmt.Println("  agres 20")
	fmt.Println("  agres -n 20")
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if opts.showVersion {
		fmt.Printf("agres %s\n", version)
		os.Exit(0)
	}

	if opts.showHelp {
		printHelp()
		os.Exit(0)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	detectors := []session.Detector{
		&agent.ClaudeDetector{},
		&agent.OpenCodeDetector{},
		&agent.AiderDetector{},
		&agent.CodexDetector{},
		&agent.AntigravityDetector{},
	}

	type result struct {
		sessions []session.Session
		err      error
	}

	results := make([]result, len(detectors))
	var wg sync.WaitGroup

	for i, d := range detectors {
		wg.Add(1)
		go func(i int, d session.Detector) {
			defer wg.Done()
			sessions, err := d.ListSessions(cwd, opts.allProjects)
			results[i] = result{sessions: sessions, err: err}
		}(i, d)
	}

	wg.Wait()

	var allSessions []session.Session
	for _, r := range results {
		if r.sessions != nil {
			allSessions = append(allSessions, r.sessions...)
		}
	}

	if len(allSessions) == 0 {
		if opts.allProjects {
			fmt.Println("No agent sessions found.")
		} else {
			fmt.Printf("No agent sessions found in: %s\n", cwd)
			fmt.Println("Run agres from a project directory that has agent sessions.")
		}
		os.Exit(0)
	}

	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].UpdatedAt.After(allSessions[j].UpdatedAt)
	})

	if len(allSessions) > opts.limit {
		allSessions = allSessions[:opts.limit]
	}

	selected, err := tui.Run(allSessions, os.Stderr, version, opts.allProjects)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if selected == nil {
		os.Exit(0)
	}

	cmd, err := resumeCommand(*selected)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot resume %s session: %v\n", selected.Agent, err)
		os.Exit(1)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start %s: %v\n", selected.Agent, err)
		os.Exit(1)
	}
}

func resumeCommand(selected session.Session) (*exec.Cmd, error) {
	if len(selected.ResumeCmd) == 0 {
		return nil, fmt.Errorf("resume command is unavailable")
	}
	if selected.WorkDir == "" {
		return nil, fmt.Errorf("original working directory is unavailable")
	}
	info, err := os.Stat(selected.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("original working directory %q: %w", selected.WorkDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("original working directory %q is not a directory", selected.WorkDir)
	}

	execPath, err := exec.LookPath(selected.ResumeCmd[0])
	if err != nil {
		execPath = selected.ResumeCmd[0]
	}
	cmd := exec.Command(execPath, selected.ResumeCmd[1:]...)
	cmd.Dir = selected.WorkDir
	return cmd, nil
}
