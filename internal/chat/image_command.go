package chat

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/media"
	"duckduckgo-chat-cli/internal/ui"

	"github.com/AlecAivazis/survey/v2"
	surveyterminal "github.com/AlecAivazis/survey/v2/terminal"
	"golang.org/x/term"
)

// ImagePromptOptions contains optional visual direction for image generation.
type ImagePromptOptions struct {
	Style  string
	Format string
}

var errImagePromptCancelled = errors.New("image generation cancelled")
var errImageReferenceCancelled = errors.New("reference image selection cancelled")

type imagePromptSurvey interface {
	Panel(title string, lines []ui.PanelLine)
	Select(message string, options []string, defaultValue string) (string, error)
	Input(message string) (string, error)
}

type terminalImagePromptSurvey struct {
	panelTitle string
	panelLines []ui.PanelLine
}

func (s *terminalImagePromptSurvey) Select(message string, options []string, defaultValue string) (string, error) {
	var choice string
	prompt := &survey.Select{Message: message, Options: options, Default: defaultValue, PageSize: 12}
	title := s.panelTitle
	lines := s.panelLines
	s.panelTitle, s.panelLines = "", nil
	if title == "" {
		title = "Image generation · " + message
	}
	err := ui.AskFramedSelect(prompt, &choice, title, lines, survey.WithStdio(os.Stdin, os.Stdout, os.Stderr), ui.ThemedSurveyIcons())
	return choice, err
}

func (s *terminalImagePromptSurvey) Panel(title string, lines []ui.PanelLine) {
	s.panelTitle = title
	s.panelLines = make([]ui.PanelLine, len(lines))
	for index, line := range lines {
		s.panelLines[index] = append(ui.PanelLine(nil), line...)
	}
}

func (terminalImagePromptSurvey) Input(message string) (string, error) {
	var value string
	err := survey.AskOne(&survey.Input{Message: message}, &value,
		survey.WithStdio(os.Stdin, os.Stdout, os.Stderr), ui.ThemedSurveyIcons())
	return value, err
}

// BuildImagePrompt composes the user's idea with any selected visual direction.
func BuildImagePrompt(idea string, options ImagePromptOptions) (string, error) {
	idea = strings.TrimSpace(idea)
	if idea == "" {
		return "", errors.New("image idea cannot be empty")
	}

	lines := []string{
		"Generate an image from the user's idea. Preserve the subjects and specific details described below.",
	}
	if value := strings.TrimSpace(options.Style); value != "" {
		lines = append(lines, "Style: "+value)
	}
	if value := strings.TrimSpace(options.Format); value != "" {
		lines = append(lines, "Preferred format: "+value)
	}
	lines = append(lines, "Idea:", idea)
	return strings.Join(lines, "\n"), nil
}

func runImageCommand(idea string, prompts imagePromptSurvey, generate func(string)) error {
	if generate == nil {
		return runImageCommandWithReference(idea, prompts, nil, nil)
	}
	return runImageCommandWithReference(idea, prompts, nil, func(prompt string, _ *media.ImageAttachment) {
		generate(prompt)
	})
}

func runImageCommandWithReference(idea string, prompts imagePromptSurvey, selectReferenceImage func() (media.ImageAttachment, error), generate func(string, *media.ImageAttachment)) error {
	if generate == nil {
		return errors.New("image generator is unavailable")
	}
	idea = strings.TrimSpace(idea)
	options := ImagePromptOptions{Style: "Photorealistic", Format: "Square (1:1)"}
	var referenceImage *media.ImageAttachment
	for {
		refLabel := "None"
		if referenceImage != nil {
			refLabel = referenceImage.Name
		}
		description := idea
		if description == "" {
			description = "Required — choose to add"
		}
		prompts.Panel("Image generation", []ui.PanelLine{
			{{Text: "Description  ", Style: ui.PanelAccent}, {Text: description, Style: ui.PanelForeground}},
			{{Text: "Style        ", Style: ui.PanelAccent}, {Text: options.Style, Style: ui.PanelForeground}},
			{{Text: "Format       ", Style: ui.PanelAccent}, {Text: options.Format, Style: ui.PanelForeground}},
			{{Text: "Reference    ", Style: ui.PanelAccent}, {Text: refLabel, Style: ui.PanelForeground}},
		})
		generateAction := "Generate image"
		if idea == "" {
			generateAction = "Description"
		}
		choices := []string{"Description", "Style", "Format", "Reference image", "Generate image", "Cancel"}
		action, err := prompts.Select("Image options", choices, generateAction)
		if err != nil {
			return err
		}
		switch action {
		case "Description":
			idea, err = prompts.Input("Describe the image")
			if err != nil {
				return err
			}
			idea = strings.TrimSpace(idea)
		case "Style":
			options.Style, err = chooseImageSetting(prompts, "Style", []string{"Photorealistic", "Illustration", "Watercolor", "3D render", "Pixel art", "Graphic design", "Custom", "Back"}, options.Style)
			if err != nil {
				return err
			}
			if options.Style == "Custom" {
				options.Style, err = prompts.Input("Describe the style")
				if err != nil {
					return err
				}
				options.Style = strings.TrimSpace(options.Style)
			}
		case "Format":
			options.Format, err = chooseImageSetting(prompts, "Format", []string{"Square (1:1)", "Portrait (4:5)", "Landscape (16:9)", "Poster (2:3)", "Custom", "Back"}, options.Format)
			if err != nil {
				return err
			}
			if options.Format == "Custom" {
				options.Format, err = prompts.Input("Aspect ratio (for example 3:2)")
				if err != nil {
					return err
				}
				options.Format = strings.TrimSpace(options.Format)
			}
		case "Reference image":
			refChoices := []string{"Choose image", "Back"}
			if referenceImage != nil {
				refChoices = []string{"Replace image", "Remove image", "Back"}
			}
			refAction, selectErr := prompts.Select("Reference image", refChoices, refChoices[0])
			if selectErr != nil {
				return selectErr
			}
			switch refAction {
			case "Choose image", "Replace image":
				if selectReferenceImage == nil {
					return errors.New("reference image selector is unavailable")
				}
				image, selectErr := selectReferenceImage()
				if errors.Is(selectErr, errImageReferenceCancelled) {
					continue
				}
				if selectErr != nil {
					return fmt.Errorf("could not prepare reference image: %w", selectErr)
				}
				referenceImage = &image
			case "Remove image":
				referenceImage = nil
			}
		case "Generate image":
			if idea == "" {
				continue
			}
			prompt, promptErr := BuildImagePrompt(idea, options)
			if promptErr != nil {
				return promptErr
			}
			if referenceImage != nil {
				prompt += "\nUse the attached reference image as visual guidance while following the written description."
			}
			lines := []ui.PanelLine{{{Text: prompt, Style: ui.PanelForeground}}}
			if referenceImage != nil {
				lines = append(lines, ui.PanelLine{{Text: "Reference: ", Style: ui.PanelAccent}, {Text: referenceImage.Name, Style: ui.PanelForeground}})
			}
			prompts.Panel("Review image", lines)
			confirm, confirmErr := prompts.Select("Ready to generate?", []string{"Generate image", "Edit settings", "Cancel"}, "Generate image")
			if confirmErr != nil {
				return confirmErr
			}
			switch confirm {
			case "Generate image":
				generate(prompt, referenceImage)
				return nil
			case "Edit settings":
				continue
			case "Cancel":
				return errImagePromptCancelled
			}
		case "Cancel":
			return errImagePromptCancelled
		default:
			return errors.New("unknown image menu action")
		}
	}
}

func chooseImageSetting(prompts imagePromptSurvey, title string, choices []string, current string) (string, error) {
	selected, err := prompts.Select(title, choices, current)
	if err != nil || selected == "Back" {
		return current, err
	}
	return selected, nil
}

// HandleImageCommand starts direct or guided image generation from the CLI.
func HandleImageCommand(c *Chat, idea string, cfg *config.Config) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		ui.Errorln("The /image prompt review requires an interactive terminal.")
		return
	}
	err := runImageCommandWithReference(idea, &terminalImagePromptSurvey{}, selectImageReference, func(prompt string, reference *media.ImageAttachment) {
		withNativeImageGeneration(c, func() {
			if reference != nil {
				c.QueueImageAttachments([]media.ImageAttachment{*reference})
			}
			logImagePrompt(prompt)
			if c.Analytics != nil {
				c.Analytics.RecordImageGenerationRequest()
			}
			ProcessInput(c, prompt, cfg)
		})
	})
	if errors.Is(err, surveyterminal.InterruptErr) || errors.Is(err, errImagePromptCancelled) {
		return
	}
	if err != nil {
		ui.Errorln("Could not prepare image request: %v", err)
	}
}

func selectImageReference() (media.ImageAttachment, error) {
	path, err := ui.SelectFile()
	if err != nil {
		return media.ImageAttachment{}, err
	}
	if path == "" {
		return media.ImageAttachment{}, errImageReferenceCancelled
	}
	return media.ReadImage(path)
}

func withNativeImageGeneration(c *Chat, run func()) {
	previousEnabled := c.NativeToolsEnabled
	previousWebSearch := c.NativeWebSearch
	previousImageGeneration := c.NativeImageGeneration
	previousSuppressDebugPayload := c.suppressDebugPayload
	defer func() {
		c.NativeToolsEnabled = previousEnabled
		c.NativeWebSearch = previousWebSearch
		c.NativeImageGeneration = previousImageGeneration
		c.suppressDebugPayload = previousSuppressDebugPayload
	}()
	c.NativeToolsEnabled = true
	c.NativeWebSearch = false
	c.NativeImageGeneration = true
	c.suppressDebugPayload = true
	run()
}

func logImagePrompt(prompt string) {
	ui.Debugln("Image generation prompt: %s", prompt)
}
