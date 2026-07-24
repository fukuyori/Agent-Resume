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

var version = "0.3.0"

func parseArgs(args []string) (limit int, showVersion bool, showHelp bool, err error) {
	limit = 10

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-v" || arg == "--version" || arg == "-version":
			showVersion = true
			return
		case arg == "-h" || arg == "--help" || arg == "-help":
			showHelp = true
			return
		case arg == "-n" || arg == "-l" || arg == "--limit" || arg == "-limit" || arg == "--number" || arg == "-number":
			if i+1 >= len(args) {
				return 0, false, false, fmt.Errorf("flag '%s' requires an integer argument", arg)
			}
			i++
			val, convErr := strconv.Atoi(args[i])
			if convErr != nil || val <= 0 {
				return 0, false, false, fmt.Errorf("invalid count '%s': must be a positive integer", args[i])
			}
			limit = val
		case strings.HasPrefix(arg, "-n=") || strings.HasPrefix(arg, "-l=") || strings.HasPrefix(arg, "--limit=") || strings.HasPrefix(arg, "-limit=") || strings.HasPrefix(arg, "--number=") || strings.HasPrefix(arg, "-number="):
			parts := strings.SplitN(arg, "=", 2)
			val, convErr := strconv.Atoi(parts[1])
			if convErr != nil || val <= 0 {
				return 0, false, false, fmt.Errorf("invalid count '%s': must be a positive integer", parts[1])
			}
			limit = val
		case !strings.HasPrefix(arg, "-"):
			val, convErr := strconv.Atoi(arg)
			if convErr == nil {
				if val <= 0 {
					return 0, false, false, fmt.Errorf("invalid count '%s': must be a positive integer", arg)
				}
				limit = val
			} else {
				return 0, false, false, fmt.Errorf("unknown argument '%s'", arg)
			}
		default:
			return 0, false, false, fmt.Errorf("unknown flag '%s'", arg)
		}
	}

	return limit, false, false, nil
}

func printHelp() {
	fmt.Printf("agres %s\n", version)
	fmt.Println("Resume CLI coding agent sessions from the current directory.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agres [options] [count]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -n, -l, --limit <count>   Number of history items to show (default: 10)")
	fmt.Println("  -v, --version             Show version information")
	fmt.Println("  -h, --help                Show help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agres")
	fmt.Println("  agres 20")
	fmt.Println("  agres -n 20")
}

func main() {
	limit, showVersion, showHelp, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if showVersion {
		fmt.Printf("agres %s\n", version)
		os.Exit(0)
	}

	if showHelp {
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
			sessions, err := d.ListSessions(cwd)
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
		fmt.Printf("No agent sessions found in: %s\n", cwd)
		fmt.Println("Run agres from a project directory that has agent sessions.")
		os.Exit(0)
	}

	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].UpdatedAt.After(allSessions[j].UpdatedAt)
	})

	if len(allSessions) > limit {
		allSessions = allSessions[:limit]
	}

	selected, err := tui.Run(allSessions, os.Stderr, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if selected == nil {
		os.Exit(0)
	}

	execPath, err := exec.LookPath(selected.ResumeCmd[0])
	if err != nil {
		execPath = selected.ResumeCmd[0]
	}

	cmd := exec.Command(execPath, selected.ResumeCmd[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start %s: %v\n", selected.Agent, err)
		os.Exit(1)
	}
}
