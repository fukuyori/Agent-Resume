package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"

	"agent-hub/internal/agent"
	"agent-hub/internal/session"
	"agent-hub/internal/tui"
)

var version = "0.1.0"

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("agres %s\n", version)
		os.Exit(0)
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
			if !d.Detect(cwd) {
				results[i] = result{}
				return
			}
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
		fmt.Println("Run agresume from a project directory that has agent sessions.")
		os.Exit(0)
	}

	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].UpdatedAt.After(allSessions[j].UpdatedAt)
	})

	selected, err := tui.Run(allSessions, os.Stderr, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if selected == nil {
		os.Exit(0)
	}

	cmd := exec.Command(selected.ResumeCmd[0], selected.ResumeCmd[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start %s: %v\n", selected.Agent, err)
		os.Exit(1)
	}
}
