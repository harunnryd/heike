package adapter

import (
	"context"
	"fmt"
	"strings"
)

type CLIAdapter struct {
	running bool
}

func NewCLIAdapter() *CLIAdapter {
	return &CLIAdapter{}
}

func (a *CLIAdapter) Name() string {
	return "cli"
}

func (a *CLIAdapter) Send(ctx context.Context, sessionID string, content string) error {
	// Simple color formatting for CLI
	// \r clears the current line (which might be the prompt "> ")
	// Then we print content
	// Then we print the prompt again

	// Check for prefixes to colorize
	color := "\033[32m" // Green for normal answers
	reset := "\033[0m"

	if strings.HasPrefix(content, "Executing:") {
		color = "\033[33m" // Yellow
	} else if strings.HasPrefix(content, "[CMD]") {
		color = "\033[34m" // Blue for slash command responses
	} else if strings.HasPrefix(content, "Plan generated") { // If we expose plans
		color = "\033[36m" // Cyan
	} else if strings.HasPrefix(content, "Error:") {
		color = "\033[31m" // Red
	} else if strings.HasPrefix(content, "Plan:") {
		color = "\033[35m" // Magenta for Plan
	}

	// Format:
	// \r<Clear Line>
	// [Heike]: <Content>
	// >

	fmt.Printf("\r\033[K") // Clear line
	fmt.Printf("%s%s%s\n", color, content, reset)
	fmt.Print("> ")

	return nil
}

func (a *CLIAdapter) Start(ctx context.Context) error {
	a.running = true
	fmt.Println("Heike CLI Adapter started. Type your message or /exit to quit.")
	fmt.Print("> ")

	go func() {
		<-ctx.Done()
		a.running = false
		fmt.Println("\nCLI Adapter stopped.")
	}()

	return nil
}

func (a *CLIAdapter) Stop(ctx context.Context) error {
	a.running = false
	return nil
}

func (a *CLIAdapter) Health(ctx context.Context) error {
	if !a.running {
		return nil // CLI adapter is always healthy even if not started
	}
	return nil
}
