package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	dir := flag.String("dir", ".", "Project directory for Claude Code")
	token := flag.String("token", "", "Auth token (or set CLAUDE_RELAY_TOKEN)")
	addr := flag.String("addr", ":8080", "Listen address")
	flag.Parse()

	if *token == "" {
		*token = os.Getenv("CLAUDE_RELAY_TOKEN")
	}
	if *token == "" {
		log.Fatal("token required: use --token or set CLAUDE_RELAY_TOKEN")
	}

	srv := NewServer(*addr, *dir, *token)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Println("\nshutting down...")
		srv.Shutdown()
	}()

	log.Printf("claude-relay listening on %s (dir: %s)", *addr, *dir)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
