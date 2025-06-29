package chat

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/fatih/color"
	"golang.org/x/term"
)

// StreamRenderer handles the progressive rendering of streaming markdown content
type StreamRenderer struct {
	renderer       *glamour.TermRenderer
	terminalWidth  int
	modelName      string
	contentStarted bool
}

// NewStreamRenderer creates a new streaming renderer
func NewStreamRenderer(modelName string) (*StreamRenderer, error) {
	width := getTerminalWidthSafe()

	customStyles := styles.DarkStyleConfig
	// H1
	customStyles.H1.StylePrimitive.Color = stringToPtr("99")
	customStyles.H1.StylePrimitive.Bold = boolToPtr(true)
	customStyles.H1.Prefix = ""
	// H2
	customStyles.H2.StylePrimitive.Color = stringToPtr("111")
	customStyles.H2.StylePrimitive.Bold = boolToPtr(true)
	customStyles.H2.Prefix = ""
	// H3
	customStyles.H3.StylePrimitive.Color = stringToPtr("118")
	customStyles.H3.StylePrimitive.Bold = boolToPtr(true)
	customStyles.H3.Prefix = ""
	// H4
	customStyles.H4.StylePrimitive.Color = stringToPtr("220")
	customStyles.H4.StylePrimitive.Bold = boolToPtr(true)
	customStyles.H4.Prefix = ""

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(customStyles),
		glamour.WithWordWrap(width-4), // Leave some margin
	)
	if err != nil {
		return nil, err
	}

	return &StreamRenderer{
		renderer:       renderer,
		terminalWidth:  width,
		modelName:      modelName,
		contentStarted: false,
	}, nil
}

// RenderStream handles the progressive rendering of a streaming response to the terminal
func RenderStream(stream <-chan string, modelName string) string {
	// Print the model name with a clear loading indicator
	color.New(color.FgHiGreen, color.Bold).Printf("%s: ", modelName)

	renderer, err := NewStreamRenderer(modelName)
	if err != nil {
		// Fallback to simple streaming
		return renderStreamFallback(stream, modelName)
	}

	return renderer.ProcessStream(stream)
}

// ProcessStream processes the incoming stream and renders it progressively
func (sr *StreamRenderer) ProcessStream(stream <-chan string) string {
	var finalContent strings.Builder

	// Show loading spinner initially
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerPos := 0

	// Create a ticker for spinner animation
	spinnerTicker := time.NewTicker(120 * time.Millisecond)
	defer spinnerTicker.Stop()

	// Show initial spinner
	fmt.Print(color.New(color.FgYellow).Sprint(spinnerChars[0]) + " ")

	// Track content display state
	contentStarted := false
	lastSpinnerUpdate := time.Now()

	// Save cursor position when content starts
	var cursorSaved bool

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				// Stream finished - replace raw content with formatted version
				sr.replaceWithFormattedContent(finalContent.String(), cursorSaved)
				return finalContent.String()
			}

			finalContent.WriteString(chunk)

			// On first chunk, clear spinner and start displaying raw content
			if !contentStarted {
				fmt.Print("\r\033[K") // Clear spinner line
				color.New(color.FgHiGreen, color.Bold).Printf("%s: ", sr.modelName)
				// Save cursor position right before content starts
				fmt.Print("\033[s") // Save cursor position
				cursorSaved = true
				contentStarted = true
			}

			// Display raw content in real-time
			fmt.Print(chunk)

		case <-spinnerTicker.C:
			// Update spinner animation only if no content has started
			if !contentStarted && time.Since(lastSpinnerUpdate) >= 100*time.Millisecond {
				spinnerPos = (spinnerPos + 1) % len(spinnerChars)
				// Update spinner in place
				fmt.Print("\r")
				color.New(color.FgHiGreen, color.Bold).Printf("%s: ", sr.modelName)
				fmt.Print(color.New(color.FgYellow).Sprint(spinnerChars[spinnerPos]) + " ")
				lastSpinnerUpdate = time.Now()
			}
		}
	}
}

// replaceWithFormattedContent clears the raw content and displays the formatted version
func (sr *StreamRenderer) replaceWithFormattedContent(content string, cursorSaved bool) {
	if content == "" {
		fmt.Println()
		return
	}

	// If cursor was saved, restore to that position and clear from there
	if cursorSaved {
		fmt.Print("\033[u") // Restore cursor position
		fmt.Print("\033[J") // Clear from cursor to end of screen
	} else {
		// Fallback: clear current line and start fresh
		fmt.Print("\r\033[K")
		color.New(color.FgHiGreen, color.Bold).Printf("%s: ", sr.modelName)
	}

	// Try to render as markdown for the final display
	rendered, err := sr.renderer.Render(content)
	if err != nil {
		// Fallback to raw text if markdown rendering fails
		fmt.Print(content)
		// Ensure we end with a newline for raw content
		if !strings.HasSuffix(content, "\n") {
			fmt.Println()
		}
	} else {
		// Print the rendered markdown
		fmt.Print(rendered)
		// Ensure we end with a newline for rendered content
		if !strings.HasSuffix(rendered, "\n") {
			fmt.Println()
		}
	}
}

// renderStreamFallback is a simple fallback when glamour fails
func renderStreamFallback(stream <-chan string, modelName string) string {
	var content strings.Builder
	var displayedLines int

	// Show simple loading
	fmt.Print(color.New(color.FgYellow).Sprint("⠋") + " ")
	contentStarted := false

	for chunk := range stream {
		content.WriteString(chunk)

		// Clear loading indicator on first chunk
		if !contentStarted {
			fmt.Print("\r\033[K") // Clear loading line
			color.New(color.FgHiGreen, color.Bold).Printf("%s: ", modelName)
			contentStarted = true
		}

		// Stream raw content in real-time
		fmt.Print(chunk)
		displayedLines += strings.Count(chunk, "\n")
	}

	// For fallback, just ensure we end with a newline (no markdown formatting)
	if !strings.HasSuffix(content.String(), "\n") {
		fmt.Println()
	}

	return content.String()
}

// getTerminalWidthSafe safely gets terminal width with fallback
func getTerminalWidthSafe() int {
	width := 80 // default

	if file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		defer file.Close()

		// Try to get terminal size
		if w, _, err := getTerminalSize(file); err == nil && w > 0 {
			width = w
		}
	}

	// Ensure reasonable bounds
	if width < 40 {
		width = 40
	} else if width > 200 {
		width = 200
	}

	return width
}

// Helper function to get terminal size
func getTerminalSize(file *os.File) (int, int, error) {
	width, height, err := term.GetSize(int(file.Fd()))
	if err != nil {
		return 80, 24, err
	}
	return width, height, nil
}

// Helper functions for style config
func stringToPtr(s string) *string {
	return &s
}

func boolToPtr(b bool) *bool {
	return &b
}
