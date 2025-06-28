package chat

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/pterm/pterm"
)

// RenderStream handles the rendering of a streaming response to the terminal.
func RenderStream(stream <-chan string, modelName string) string {
	var responseBuilder strings.Builder
	pterm.Print(pterm.LightGreen(modelName + ": "))

	// Start an area that can be overwritten.
	area, _ := pterm.DefaultArea.Start()
	defer area.Stop()

	// glamour renderer
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating renderer: %v\n", err)
		// Fallback to simple printing
		for chunk := range stream {
			fmt.Print(chunk)
			responseBuilder.WriteString(chunk)
		}
		return responseBuilder.String()
	}

	for chunk := range stream {
		responseBuilder.WriteString(chunk)
		out, err := renderer.Render(responseBuilder.String())
		if err != nil {
			// On render error, fallback to printing the raw chunk
			area.Update(responseBuilder.String())
			continue
		}
		area.Update(out)
	}

	// Final render to clean up any glamour artifacts.
	finalOutput, _ := renderer.Render(responseBuilder.String())
	area.Update(finalOutput)

	return responseBuilder.String()
}
