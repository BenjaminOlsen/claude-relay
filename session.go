package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// Session tracks a Claude Code conversation across multiple process invocations.
// Each Send() spawns a new claude process with --resume to maintain context.
type Session struct {
	dir       string
	sessionID string
	mu        sync.Mutex
	busy      bool
}

func NewSession(dir string) *Session {
	return &Session{dir: dir}
}

// Send spawns a claude process for the given message and calls onLine for each
// stdout line. The process exits when the response is complete.
func (s *Session) Send(message string, onLine func(line []byte)) error {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return fmt.Errorf("session busy")
	}
	s.busy = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
	}()

	args := []string{"-p", "--verbose", "--output-format", "stream-json"}
	if s.sessionID != "" {
		args = append(args, "--resume", s.sessionID)
	}
	args = append(args, message)

	cmd := exec.Command("claude", args...)
	cmd.Dir = s.dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	log.Printf("claude started (pid %d, session %s)", cmd.Process.Pid, s.sessionID)

	// Log stderr
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			log.Printf("[claude stderr] %s", sc.Text())
		}
	}()

	// Read stdout line by line
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		out := make([]byte, len(line))
		copy(out, line)

		// Extract session_id from any event
		if s.sessionID == "" {
			s.extractSessionID(out)
		}

		onLine(out)
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("claude exited: %v", err)
	}

	return nil
}

func (s *Session) extractSessionID(line []byte) {
	var evt struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(line, &evt) == nil && evt.SessionID != "" {
		s.sessionID = evt.SessionID
		log.Printf("session id: %s", s.sessionID)
	}
}
