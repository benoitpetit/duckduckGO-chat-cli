package chat

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"duckduckgo-chat-cli/internal/media"
	"duckduckgo-chat-cli/internal/ui"

	"github.com/fatih/color"
)

type imageSurveyCall struct {
	kind         string
	message      string
	options      []string
	defaultValue string
}

type fakeImagePromptSurvey struct {
	selections  []string
	inputs      []string
	selectErrAt int
	inputErrAt  int
	err         error
	calls       []imageSurveyCall
	panels      []imageSurveyPanel
}

type imageSurveyPanel struct {
	title string
	lines []ui.PanelLine
}

func (f *fakeImagePromptSurvey) Panel(title string, lines []ui.PanelLine) {
	cloned := make([]ui.PanelLine, len(lines))
	for i := range lines {
		cloned[i] = append(ui.PanelLine(nil), lines[i]...)
	}
	f.panels = append(f.panels, imageSurveyPanel{title: title, lines: cloned})
}

func (f *fakeImagePromptSurvey) Select(message string, options []string, defaultValue string) (string, error) {
	callIndex := len(f.calls)
	f.calls = append(f.calls, imageSurveyCall{kind: "select", message: message, options: append([]string(nil), options...), defaultValue: defaultValue})
	if f.err != nil && f.selectErrAt == callIndex {
		return "", f.err
	}
	selectionIndex := 0
	for _, call := range f.calls {
		if call.kind == "select" {
			selectionIndex++
		}
	}
	selectionIndex--
	if selectionIndex >= len(f.selections) {
		return "", errors.New("no fake selection available")
	}
	return f.selections[selectionIndex], nil
}

func (f *fakeImagePromptSurvey) Input(message string) (string, error) {
	callIndex := len(f.calls)
	f.calls = append(f.calls, imageSurveyCall{kind: "input", message: message})
	if f.err != nil && f.inputErrAt == callIndex {
		return "", f.err
	}
	inputIndex := 0
	for _, call := range f.calls {
		if call.kind == "input" {
			inputIndex++
		}
	}
	inputIndex--
	if inputIndex >= len(f.inputs) {
		return "", errors.New("no fake input available")
	}
	return f.inputs[inputIndex], nil
}

func TestBuildImagePromptUsesEssentialOptions(t *testing.T) {
	prompt, err := BuildImagePrompt("  a blue bird carrying a tiny lantern  ", ImagePromptOptions{Style: "Watercolor", Format: "Portrait (4:5)"})
	if err != nil {
		t.Fatalf("BuildImagePrompt() error = %v", err)
	}
	for _, want := range []string{"a blue bird carrying a tiny lantern", "Style: Watercolor", "Preferred format: Portrait (4:5)"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt %q missing %q", prompt, want)
		}
	}
	if strings.Contains(prompt, "Scene / background") || strings.Contains(prompt, "Typography:") {
		t.Fatalf("obsolete questionnaire details leaked into prompt: %q", prompt)
	}
}

func TestBuildImagePromptRejectsBlankIdea(t *testing.T) {
	if _, err := BuildImagePrompt(" \n\t ", ImagePromptOptions{}); err == nil {
		t.Fatal("BuildImagePrompt() accepted a blank idea")
	}
}

func TestRunImageCommandStartsWithCompactSettingsMenu(t *testing.T) {
	fake := &fakeImagePromptSurvey{selections: []string{"Generate image", "Generate image"}}
	var generated string
	if err := runImageCommand("A fox under a pine tree", fake, func(prompt string) { generated = prompt }); err != nil {
		t.Fatalf("runImageCommand() error = %v", err)
	}
	if !strings.Contains(generated, "A fox under a pine tree") {
		t.Fatalf("generated prompt = %q", generated)
	}
	if len(fake.panels) < 2 || fake.panels[0].title != "Image generation" || fake.panels[1].title != "Review image" {
		t.Fatalf("panels = %+v", fake.panels)
	}
	want := []string{"Description", "Style", "Format", "Reference image", "Generate image", "Cancel"}
	if !reflect.DeepEqual(fake.calls[0].options, want) {
		t.Fatalf("main menu options = %v, want %v", fake.calls[0].options, want)
	}
}

func TestRunImageCommandDescriptionIsCollectedOnlyWhenNeeded(t *testing.T) {
	fake := &fakeImagePromptSurvey{selections: []string{"Description", "Generate image", "Generate image"}, inputs: []string{"A fox reading beside a window"}}
	var generated string
	if err := runImageCommand("", fake, func(prompt string) { generated = prompt }); err != nil {
		t.Fatalf("runImageCommand() error = %v", err)
	}
	if !strings.Contains(generated, "A fox reading beside a window") {
		t.Fatalf("generated prompt = %q", generated)
	}
	if len(fake.calls) != 4 || fake.calls[1].message != "Describe the image" {
		t.Fatalf("calls = %+v", fake.calls)
	}
}

func TestRunImageCommandSettingsUseSubmenusAndReturnToMainMenu(t *testing.T) {
	fake := &fakeImagePromptSurvey{selections: []string{"Style", "Illustration", "Format", "Landscape (16:9)", "Generate image", "Generate image"}}
	var generated string
	if err := runImageCommand("A fox", fake, func(prompt string) { generated = prompt }); err != nil {
		t.Fatalf("runImageCommand() error = %v", err)
	}
	if !strings.Contains(generated, "Style: Illustration") || !strings.Contains(generated, "Preferred format: Landscape (16:9)") {
		t.Fatalf("generated prompt = %q", generated)
	}
	if fake.calls[1].message != "Style" || fake.calls[3].message != "Format" {
		t.Fatalf("submenu calls = %+v", fake.calls)
	}
}

func TestRunImageCommandReferenceImageIsOptionalAndAttached(t *testing.T) {
	reference := media.ImageAttachment{Name: "style-guide.webp", MIMEType: "image/webp", Data: []byte("reference bytes")}
	fake := &fakeImagePromptSurvey{selections: []string{"Reference image", "Choose image", "Generate image", "Generate image"}}
	var generatedPrompt string
	var generatedReference *media.ImageAttachment
	selectCalls := 0
	err := runImageCommandWithReference("A fox", fake, func() (media.ImageAttachment, error) { selectCalls++; return reference, nil }, func(prompt string, image *media.ImageAttachment) { generatedPrompt, generatedReference = prompt, image })
	if err != nil {
		t.Fatalf("runImageCommandWithReference() error = %v", err)
	}
	if selectCalls != 1 || generatedReference == nil || generatedReference.Name != reference.Name || string(generatedReference.Data) != string(reference.Data) {
		t.Fatalf("reference selector calls=%d image=%+v", selectCalls, generatedReference)
	}
	if !strings.Contains(generatedPrompt, "A fox") || !strings.Contains(generatedPrompt, "attached reference image") {
		t.Fatalf("generated prompt = %q", generatedPrompt)
	}
	if !strings.Contains(fake.panels[len(fake.panels)-1].lines[1][1].Text, "style-guide.webp") {
		t.Fatalf("reference absent from review panel: %+v", fake.panels[len(fake.panels)-1])
	}
}

func TestRunImageCommandReferenceCanBeRemoved(t *testing.T) {
	fake := &fakeImagePromptSurvey{selections: []string{"Reference image", "Choose image", "Reference image", "Remove image", "Generate image", "Generate image"}}
	images := []media.ImageAttachment{{Name: "style.webp", Data: []byte("ref")}}
	selected := 0
	var got *media.ImageAttachment
	err := runImageCommandWithReference("A fox", fake, func() (media.ImageAttachment, error) { image := images[selected]; selected++; return image, nil }, func(_ string, image *media.ImageAttachment) { got = image })
	if err != nil {
		t.Fatalf("runImageCommandWithReference() error = %v", err)
	}
	if selected != 1 || got != nil {
		t.Fatalf("selected=%d final reference=%+v", selected, got)
	}
}

func TestRunImageCommandReferencePickerCancelReturnsToMenu(t *testing.T) {
	fake := &fakeImagePromptSurvey{selections: []string{"Reference image", "Choose image", "Generate image", "Generate image"}}
	generated := false
	err := runImageCommandWithReference("A fox", fake, func() (media.ImageAttachment, error) { return media.ImageAttachment{}, errImageReferenceCancelled }, func(_ string, image *media.ImageAttachment) { generated = image == nil })
	if err != nil || !generated {
		t.Fatalf("err=%v generated=%t", err, generated)
	}
}

func TestRunImageCommandReviewCanReturnToSettings(t *testing.T) {
	fake := &fakeImagePromptSurvey{selections: []string{"Generate image", "Edit settings", "Style", "Watercolor", "Generate image", "Generate image"}}
	var generated string
	err := runImageCommand("A fox", fake, func(prompt string) { generated = prompt })
	if err != nil {
		t.Fatalf("runImageCommand() error = %v", err)
	}
	if !strings.Contains(generated, "Style: Watercolor") {
		t.Fatalf("generated prompt = %q", generated)
	}
}

func TestRunImageCommandCancellationDoesNotGenerate(t *testing.T) {
	fake := &fakeImagePromptSurvey{selections: []string{"Cancel"}}
	generated := false
	err := runImageCommand("A fox", fake, func(string) { generated = true })
	if !errors.Is(err, errImagePromptCancelled) || generated {
		t.Fatalf("err=%v generated=%t", err, generated)
	}
}

func TestRunImageCommandPropagatesSurveyInterrupt(t *testing.T) {
	interrupt := errors.New("survey interrupted")
	fake := &fakeImagePromptSurvey{selectErrAt: 0, err: interrupt}
	if err := runImageCommand("A fox", fake, func(string) {}); !errors.Is(err, interrupt) {
		t.Fatalf("error=%v, want interrupt", err)
	}
}

func TestWithNativeImageGenerationRestoresSettings(t *testing.T) {
	chat := &Chat{NativeToolsEnabled: false, NativeWebSearch: true, NativeImageGeneration: false}
	called := false
	withNativeImageGeneration(chat, func() {
		called = true
		if !chat.NativeToolsEnabled || chat.NativeWebSearch || !chat.NativeImageGeneration {
			t.Fatalf("temporary native tool settings = enabled:%t web:%t image:%t", chat.NativeToolsEnabled, chat.NativeWebSearch, chat.NativeImageGeneration)
		}
	})
	if !called {
		t.Fatal("generation callback was not called")
	}
	if chat.NativeToolsEnabled || !chat.NativeWebSearch || chat.NativeImageGeneration {
		t.Fatalf("native tool settings after success = enabled:%t web:%t image:%t", chat.NativeToolsEnabled, chat.NativeWebSearch, chat.NativeImageGeneration)
	}
}

func TestWithNativeImageGenerationRestoresSettingsAfterPanic(t *testing.T) {
	chat := &Chat{NativeToolsEnabled: true, NativeWebSearch: true, NativeImageGeneration: false}
	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("generation callback did not panic")
			}
		}()
		withNativeImageGeneration(chat, func() {
			if !chat.NativeToolsEnabled || chat.NativeWebSearch || !chat.NativeImageGeneration {
				t.Fatalf("temporary native tool settings = enabled:%t web:%t image:%t", chat.NativeToolsEnabled, chat.NativeWebSearch, chat.NativeImageGeneration)
			}
			panic("generation failed")
		})
	}()
	if !chat.NativeToolsEnabled || !chat.NativeWebSearch || chat.NativeImageGeneration {
		t.Fatalf("native tool settings after panic = enabled:%t web:%t image:%t", chat.NativeToolsEnabled, chat.NativeWebSearch, chat.NativeImageGeneration)
	}
}

func TestImageCommandSuppressesPayloadDebug(t *testing.T) {
	t.Setenv("DEBUG", "true")
	chat := &Chat{}
	if !shouldLogRequestPayload(chat) {
		t.Fatal("ordinary chat request payload should remain available in debug mode")
	}
	withNativeImageGeneration(chat, func() {
		if shouldLogRequestPayload(chat) {
			t.Fatal("image request payload would duplicate the separately logged image prompt")
		}
	})
	if !shouldLogRequestPayload(chat) {
		t.Fatal("request payload debug setting was not restored after image generation")
	}
}

func TestImagePromptDebugVisibility(t *testing.T) {
	const prompt = "Generate an image from the user's idea: a silver moon"
	for _, test := range []struct {
		name      string
		debug     string
		wantCount int
	}{
		{name: "normal mode", debug: "", wantCount: 0},
		{name: "debug mode", debug: "true", wantCount: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DEBUG", test.debug)
			output := captureImageCommandStdout(t, func() { logImagePrompt(prompt) })
			if got := strings.Count(output, prompt); got != test.wantCount {
				t.Fatalf("prompt appeared %d times in output, want %d; output=%q", got, test.wantCount, output)
			}
			if test.wantCount == 1 && !strings.Contains(output, "Image generation prompt:") {
				t.Fatalf("debug output is missing the image prompt label: %q", output)
			}
		})
	}
}

func captureImageCommandStdout(t *testing.T, run func()) string {
	t.Helper()
	originalStdout := os.Stdout
	originalColorOutput := color.Output
	outputFile, err := os.CreateTemp(t.TempDir(), "image-command-output-*")
	if err != nil {
		t.Fatal(err)
	}
	defer outputFile.Close()
	os.Stdout = outputFile
	color.Output = outputFile
	defer func() {
		os.Stdout = originalStdout
		color.Output = originalColorOutput
	}()
	run()
	if _, err := outputFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
