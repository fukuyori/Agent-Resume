package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"agres/internal/agent"
	"agres/internal/clean"
	"agres/internal/session"
	"agres/internal/tui"
)

var version = "0.5.1"

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
	fmt.Println("  agres clean [clean options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -a, --all                 Show sessions from all projects")
	fmt.Println("  -n, -l, --limit <count>   Number of history items to show (default: 10)")
	fmt.Println("  -v, --version             Show version information")
	fmt.Println("  -h, --help                Show help message")
	fmt.Println()
	fmt.Println("Clean options (agres clean):")
	fmt.Println("  -a, --all                 Clean sessions from all projects")
	fmt.Println("  --older-than <dur>        Only sessions updated before <dur> ago (default: 30d; 0 disables)")
	fmt.Println("  --larger-than <size>      Only sessions whose history is at least <size> (e.g. 10M)")
	fmt.Println("  --agent <name>            Only sessions of one agent (claude, codex, opencode, agy)")
	fmt.Println("  --keep <count>            Always keep the newest <count> sessions per project (default: 3)")
	fmt.Println("  -y, --yes                 Delete without asking")
	fmt.Println("  --dry-run                 Show what would be deleted and exit")
	fmt.Println("  Sessions updated within the last hour are never deleted. Aider is not supported.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agres")
	fmt.Println("  agres --all")
	fmt.Println("  agres -a --limit 20")
	fmt.Println("  agres 20")
	fmt.Println("  agres -n 20")
	fmt.Println("  agres clean")
	fmt.Println("  agres clean -a --larger-than 10M --older-than 0")
}

type cleanOptions struct {
	clean.Options
	allProjects bool
	yes         bool
	dryRun      bool
}

func parseCleanArgs(args []string) (cleanOptions, error) {
	opts := cleanOptions{Options: clean.DefaultOptions()}
	next := func(i *int, flag string) (string, error) {
		if *i+1 >= len(args) {
			return "", fmt.Errorf("flag '%s' requires an argument", flag)
		}
		*i++
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, inline, hasInline := strings.Cut(arg, "=")
		value := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			return next(&i, name)
		}
		var err error
		switch name {
		case "-a", "--all":
			opts.allProjects = true
		case "-y", "--yes":
			opts.yes = true
		case "--dry-run":
			opts.dryRun = true
		case "--older-than":
			var v string
			if v, err = value(); err == nil {
				opts.OlderThan, err = clean.ParseDuration(v)
			}
		case "--larger-than":
			var v string
			if v, err = value(); err == nil {
				opts.LargerThan, err = clean.ParseSize(v)
			}
		case "--keep":
			var v string
			if v, err = value(); err == nil {
				opts.Keep, err = strconv.Atoi(v)
				if err != nil || opts.Keep < 0 {
					err = fmt.Errorf("invalid keep count '%s'", v)
				}
			}
		case "--agent":
			var v string
			if v, err = value(); err == nil {
				opts.Agent = session.Agent(strings.ToLower(v))
				switch opts.Agent {
				case session.AgentClaude, session.AgentCodex, session.AgentOpenCode, session.AgentAntigravity:
				case "antigravity":
					opts.Agent = session.AgentAntigravity
				default:
					err = fmt.Errorf("unknown agent '%s'", v)
				}
			}
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		default:
			err = fmt.Errorf("unknown flag '%s'", arg)
		}
		if err != nil {
			return cleanOptions{}, err
		}
	}
	return opts, nil
}

func runClean(args []string) {
	opts, err := parseCleanArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cleaners := []session.Cleaner{
		&agent.ClaudeDetector{},
		&agent.OpenCodeDetector{},
		&agent.CodexDetector{},
		&agent.AntigravityDetector{},
	}
	owner := make(map[string]session.Cleaner)
	var all []session.Session
	for _, c := range cleaners {
		found, _ := c.ListSessions(cwd, opts.allProjects)
		for _, s := range found {
			owner[string(s.Agent)+"\x00"+s.ID] = c
		}
		all = append(all, found...)
	}

	targets := clean.Select(all, opts.Options, time.Now())
	if len(targets) == 0 {
		fmt.Println("Nothing to clean.")
		return
	}

	var total int64
	for _, s := range targets {
		total += s.Size
		project := s.WorkDir
		if !opts.allProjects {
			project = ""
		}
		fmt.Printf("  %s  %-10s %7s  %s%s\n",
			s.UpdatedAt.Local().Format("2006-01-02 15:04"),
			"["+string(s.Agent)+"]",
			clean.FormatSize(s.Size),
			projectPrefix(project),
			s.Title)
	}
	fmt.Println()
	if opts.dryRun {
		fmt.Printf("%d sessions (%s) would be deleted.\n", len(targets), clean.FormatSize(total))
		return
	}
	if !opts.yes {
		fmt.Printf("Delete %d sessions (%s)? [y/N] ", len(targets), clean.FormatSize(total))
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if ans := strings.ToLower(strings.TrimSpace(line)); ans != "y" && ans != "yes" {
			fmt.Println("Aborted.")
			return
		}
	}

	deleted := 0
	var freed int64
	for _, s := range targets {
		c := owner[string(s.Agent)+"\x00"+s.ID]
		if err := c.Delete(s); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete %s %s: %v\n", s.Agent, s.ID, err)
			continue
		}
		deleted++
		freed += s.Size
	}
	fmt.Printf("Deleted %d sessions (%s).\n", deleted, clean.FormatSize(freed))
}

func projectPrefix(workDir string) string {
	if workDir == "" {
		return ""
	}
	return "[" + workDir + "]  "
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "clean" {
		runClean(os.Args[2:])
		return
	}

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
