package chat

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/ui"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

type displayEntry struct {
	user           bool
	model          string
	markdown       string
	framedOverride *bool
	panelTitle     string
	panelLines     []ui.PanelLine
}

// ConversationView retains exactly the text needed to lay out the visible
// conversation again after the terminal changes size. It is only used by the
// interactive CLI; the network conversation has its own history.
type ConversationView struct {
	mu            sync.Mutex
	header        string
	help          string
	framed        bool
	entries       []displayEntry
	suspendRedraw int
	pendingWidth  int
}

func NewConversationView(header, help string, framed bool) *ConversationView {
	return &ConversationView{header: header, help: help, framed: framed}
}

func (v *ConversationView) SetHeader(header string) {
	if v == nil {
		return
	}
	v.mu.Lock()
	v.header = header
	v.mu.Unlock()
}

func (v *ConversationView) SetFramed(framed bool) {
	if v == nil {
		return
	}
	v.mu.Lock()
	v.framed = framed
	v.mu.Unlock()
}

func (v *ConversationView) RecordUser(input string) {
	if v == nil || strings.TrimSpace(input) == "" {
		return
	}
	v.mu.Lock()
	v.entries = append(v.entries, displayEntry{user: true, markdown: input})
	v.mu.Unlock()
}

func (v *ConversationView) RecordAssistant(model, markdown string) {
	if v == nil || strings.TrimSpace(markdown) == "" {
		return
	}
	v.mu.Lock()
	v.entries = append(v.entries, displayEntry{model: model, markdown: markdown})
	v.mu.Unlock()
}

// RecordAssistantWithFrame retains a per-answer frame choice for later redraws.
func (v *ConversationView) RecordAssistantWithFrame(model, markdown string, framed bool) {
	if v == nil || strings.TrimSpace(markdown) == "" {
		return
	}
	v.mu.Lock()
	v.entries = append(v.entries, displayEntry{model: model, markdown: markdown, framedOverride: &framed})
	v.mu.Unlock()
}

// RecordPanel adds a rendered UI panel to the retained screen history.
func (v *ConversationView) RecordPanel(title string, lines []ui.PanelLine) {
	if v == nil {
		return
	}
	cloned := make([]ui.PanelLine, len(lines))
	for i, line := range lines {
		cloned[i] = append(ui.PanelLine(nil), line...)
	}
	v.mu.Lock()
	v.entries = append(v.entries, displayEntry{panelTitle: title, panelLines: cloned})
	v.mu.Unlock()
}

// RecordSourceReferences retains compact, path-safe references for a request.
func (v *ConversationView) RecordSourceReferences(sources []chatcontext.SourceRef) []ui.PanelLine {
	if v == nil || len(sources) == 0 {
		return nil
	}
	lines := make([]ui.PanelLine, 0, len(sources))
	for _, source := range sources {
		state := source.State
		if state == "" {
			state = chatcontext.SourceStateActive
		}
		kind := source.Kind
		if kind == "" {
			kind = "unknown"
		}
		lines = append(lines, ui.PanelLine{{Text: fmt.Sprintf("%s · %s · %s · %s · %s", source.ID, kind, displaySourceName(source), approximateSizeLabel(source.SizeBytes), state), Style: ui.PanelInfo}})
	}
	v.RecordPanel("Sources for this exchange", lines)
	return lines
}

// UpdatePanel replaces the most recently retained panel with the same title,
// or records it when no matching panel exists yet.
func (v *ConversationView) UpdatePanel(title string, lines []ui.PanelLine) {
	if v == nil {
		return
	}
	cloned := make([]ui.PanelLine, len(lines))
	for i, line := range lines {
		cloned[i] = append(ui.PanelLine(nil), line...)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	for i := len(v.entries) - 1; i >= 0; i-- {
		if v.entries[i].panelTitle == title {
			v.entries[i].panelLines = cloned
			return
		}
	}
	v.entries = append(v.entries, displayEntry{panelTitle: title, panelLines: cloned})
}

// SuspendRedraw prevents a resize callback from clearing an active interactive
// view such as the configuration selector. The returned function resumes it.
func (v *ConversationView) SuspendRedraw() func() {
	if v == nil {
		return func() {}
	}
	v.mu.Lock()
	v.suspendRedraw++
	v.mu.Unlock()
	return func() {
		width := 0
		v.mu.Lock()
		if v.suspendRedraw > 0 {
			v.suspendRedraw--
		}
		if v.suspendRedraw == 0 && v.pendingWidth > 0 {
			width = v.pendingWidth
			v.pendingWidth = 0
		}
		v.mu.Unlock()
		if width > 0 {
			v.Redraw(width)
		}
	}
}

func (v *ConversationView) Clear() {
	if v == nil {
		return
	}
	v.mu.Lock()
	v.entries = nil
	v.mu.Unlock()
}

// Redraw clears the visible screen only. It deliberately keeps terminal
// scrollback, which may still contain the older layout.
func (v *ConversationView) Redraw(width int) {
	if v == nil || !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	v.mu.Lock()
	if v.suspendRedraw > 0 {
		if width > 0 {
			v.pendingWidth = width
		}
		v.mu.Unlock()
		return
	}
	entries := append([]displayEntry(nil), v.entries...)
	header, help, framed := v.header, v.help, v.framed
	for i := range entries {
		if entries[i].panelLines != nil {
			lines := make([]ui.PanelLine, len(entries[i].panelLines))
			for j, line := range entries[i].panelLines {
				lines[j] = append(ui.PanelLine(nil), line...)
			}
			entries[i].panelLines = lines
		}
	}
	v.mu.Unlock()
	fmt.Print("\x1b[2J\x1b[H")
	if header != "" {
		fmt.Println(header)
	}
	if help != "" {
		fmt.Println(help)
	}
	for _, entry := range entries {
		if entry.panelTitle != "" {
			ui.PrintStyledPanelAtWidth(entry.panelTitle, entry.panelLines, width)
			continue
		}
		if entry.user {
			printDisplayUser(entry.markdown, width)
			continue
		}
		entryFramed := framed
		if entry.framedOverride != nil {
			entryFramed = *entry.framedOverride
		}
		renderMarkdownAtWidth(entry.markdown, entry.model, width, entryFramed)
	}
}

func printDisplayUser(input string, width int) {
	if width < 8 {
		width = 8
	}
	for index, part := range strings.Split(input, "\n") {
		part = normalizeFrameLine(part)
		if index == 0 {
			part = "You: " + part
		}
		fmt.Println(ansi.Hardwrap(part, width, true))
	}
}
