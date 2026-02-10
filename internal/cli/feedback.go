package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	// Colors for different types of output
	successColor = color.New(color.FgGreen)
	errorColor   = color.New(color.FgRed)
	warnColor    = color.New(color.FgYellow)
	infoColor    = color.New(color.FgCyan)
	boldColor    = color.New(color.Bold)
)

// Feedback provides methods for consistent CLI feedback
type Feedback struct {
	verbose bool
}

// NewFeedback creates a new Feedback instance
func NewFeedback(verbose bool) *Feedback {
	return &Feedback{
		verbose: verbose,
	}
}

// Success prints a success message
func (f *Feedback) Success(format string, a ...interface{}) {
	successColor.Printf("✓ ")
	fmt.Printf(format+"\n", a...)
}

// Error prints an error message
func (f *Feedback) Error(format string, a ...interface{}) {
	errorColor.Printf("✗ ")
	fmt.Printf(format+"\n", a...)
}

// Warning prints a warning message
func (f *Feedback) Warning(format string, a ...interface{}) {
	warnColor.Printf("! ")
	fmt.Printf(format+"\n", a...)
}

// Info prints an info message
func (f *Feedback) Info(format string, a ...interface{}) {
	if f.verbose {
		infoColor.Printf("ℹ ")
		fmt.Printf(format+"\n", a...)
	}
}

// StartSpinner starts a loading spinner with a message
func (f *Feedback) StartSpinner(message string) *Spinner {
	return NewSpinner(message)
}

// PrintHeader prints a header for a command
func (f *Feedback) PrintHeader(command string) {
	boldColor.Printf("\nMeridian - %s\n", strings.Title(command))
	fmt.Println(strings.Repeat("-", 40))
}

// PrintSummary prints a summary of an operation
func (f *Feedback) PrintSummary(success bool, duration time.Duration, details string) {
	fmt.Println(strings.Repeat("-", 40))
	if success {
		f.Success("Operation completed in %s", duration.Round(time.Millisecond))
		if details != "" {
			fmt.Println(details)
		}
	} else {
		f.Error("Operation failed after %s", duration.Round(time.Millisecond))
		if details != "" {
			fmt.Println(details)
		}
	}
}

// ExitWithError prints an error message and exits with code 1
func (f *Feedback) ExitWithError(format string, a ...interface{}) {
	f.Error(format, a...)
	os.Exit(1)
}

// Spinner represents a loading spinner
type Spinner struct {
	message string
	stop    chan struct{}
	done    chan struct{}
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	s := &Spinner{
		message: message,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go s.spin()
	return s
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
}

// StopWithSuccess stops the spinner with a success message
func (s *Spinner) StopWithSuccess(format string, a ...interface{}) {
	s.Stop()
	successColor.Printf("✓ ")
	fmt.Printf(format+"\n", a...)
}

// StopWithError stops the spinner with an error message
func (s *Spinner) StopWithError(format string, a ...interface{}) {
	s.Stop()
	errorColor.Printf("✗ ")
	fmt.Printf(format+"\n", a...)
}

func (s *Spinner) spin() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	defer close(s.done)

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			fmt.Printf("\r%s %s", frames[i], s.message)
			i = (i + 1) % len(frames)
		}
	}
} 