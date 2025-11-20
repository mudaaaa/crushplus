package editor

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mudaaaa/crushplus/internal/app"
	"github.com/mudaaaa/crushplus/internal/fsext"
	"github.com/mudaaaa/crushplus/internal/message"
	"github.com/mudaaaa/crushplus/internal/session"
	"github.com/mudaaaa/crushplus/internal/tui/components/chat"
	"github.com/mudaaaa/crushplus/internal/tui/components/completions"
	"github.com/mudaaaa/crushplus/internal/tui/components/core/layout"
	"github.com/mudaaaa/crushplus/internal/tui/components/dialogs"
	"github.com/mudaaaa/crushplus/internal/tui/components/dialogs/commands"
	"github.com/mudaaaa/crushplus/internal/tui/components/dialogs/filepicker"
	"github.com/mudaaaa/crushplus/internal/tui/components/dialogs/quit"
	"github.com/mudaaaa/crushplus/internal/tui/styles"
	"github.com/mudaaaa/crushplus/internal/tui/util"
)

type Editor interface {
	util.Model
	layout.Sizeable
	layout.Focusable
	layout.Help
	layout.Positional

	SetSession(session session.Session) tea.Cmd
	IsCompletionsOpen() bool
	HasAttachments() bool
	Cursor() *tea.Cursor
}

type FileCompletionItem struct {
	Path string // The file path
}

type editorCmp struct {
	width              int
	height             int
	x, y               int
	app                *app.App
	session            session.Session
	textarea           textarea.Model
	attachments        []message.Attachment
	deleteMode         bool
	readyPlaceholder   string
	workingPlaceholder string
	shimmerOffset      float64 // For animating placeholder shimmer effect

	keyMap EditorKeyMap

	// File path completions
	currentQuery          string
	completionsStartIndex int
	isCompletionsOpen     bool
}

var DeleteKeyMaps = DeleteAttachmentKeyMaps{
	AttachmentDeleteMode: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r+{i}", "delete attachment at index i"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc", "alt+esc"),
		key.WithHelp("esc", "cancel delete mode"),
	),
	DeleteAllAttachments: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("ctrl+r+r", "delete all attachments"),
	),
}

const (
	maxAttachments = 5
	maxFileResults = 25
)

type OpenEditorMsg struct {
	Text string
}

type shimmerTickMsg struct{}


func (m *editorCmp) openEditor(value string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Use platform-appropriate default editor
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "nvim"
		}
	}

	tmpfile, err := os.CreateTemp("", "msg_*.md")
	if err != nil {
		return util.ReportError(err)
	}
	defer tmpfile.Close() //nolint:errcheck
	if _, err := tmpfile.WriteString(value); err != nil {
		return util.ReportError(err)
	}
	c := exec.CommandContext(context.TODO(), editor, tmpfile.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return util.ReportError(err)
		}
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return util.ReportError(err)
		}
		if len(content) == 0 {
			return util.ReportWarn("Message is empty")
		}
		os.Remove(tmpfile.Name())
		return OpenEditorMsg{
			Text: strings.TrimSpace(string(content)),
		}
	})
}

func (m *editorCmp) Init() tea.Cmd {
	return m.shimmerTick()
}

func (m *editorCmp) shimmerTick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(time.Time) tea.Msg {
		return shimmerTickMsg{}
	})
}

func (m *editorCmp) send() tea.Cmd {
	value := m.textarea.Value()
	value = strings.TrimSpace(value)

	switch value {
	case "exit", "quit":
		m.textarea.Reset()
		return util.CmdHandler(dialogs.OpenDialogMsg{Model: quit.NewQuitDialog()})
	}

	m.textarea.Reset()
	attachments := m.attachments

	m.attachments = nil
	if value == "" {
		return nil
	}

	// Change the placeholder when sending a new message.
	m.randomizePlaceholders()

	return tea.Batch(
		util.CmdHandler(chat.SendMsg{
			Text:        value,
			Attachments: attachments,
		}),
	)
}

func (m *editorCmp) repositionCompletions() tea.Msg {
	x, y := m.completionsPosition()
	return completions.RepositionCompletionsMsg{X: x, Y: y}
}

func (m *editorCmp) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case shimmerTickMsg:
		// Update shimmer offset for placeholder animation
		m.shimmerOffset += 0.05
		if m.shimmerOffset \u003e 1.0 {
			m.shimmerOffset = 0.0
		}
		return m, m.shimmerTick()
	case tea.WindowSizeMsg:
		return m, m.repositionCompletions
	case filepicker.FilePickedMsg:
		if len(m.attachments) >= maxAttachments {
			return m, util.ReportError(fmt.Errorf("cannot add more than %d images", maxAttachments))
		}
		m.attachments = append(m.attachments, msg.Attachment)
		return m, nil
	case completions.CompletionsOpenedMsg:
		m.isCompletionsOpen = true
	case completions.CompletionsClosedMsg:
		m.isCompletionsOpen = false
		m.currentQuery = ""
		m.completionsStartIndex = 0
	case completions.SelectCompletionMsg:
		if !m.isCompletionsOpen {
			return m, nil
		}
		if item, ok := msg.Value.(FileCompletionItem); ok {
			word := m.textarea.Word()
			// If the selected item is a file, insert its path into the textarea
			value := m.textarea.Value()
			value = value[:m.completionsStartIndex] + // Remove the current query
				item.Path + // Insert the file path
				value[m.completionsStartIndex+len(word):] // Append the rest of the value
			// XXX: This will always move the cursor to the end of the textarea.
			m.textarea.SetValue(value)
			m.textarea.MoveToEnd()
			if !msg.Insert {
				m.isCompletionsOpen = false
				m.currentQuery = ""
				m.completionsStartIndex = 0
			}
		}

	case commands.OpenExternalEditorMsg:
		if m.app.AgentCoordinator.IsSessionBusy(m.session.ID) {
			return m, util.ReportWarn("Agent is working, please wait...")
		}
		return m, m.openEditor(m.textarea.Value())
	case OpenEditorMsg:
		m.textarea.SetValue(msg.Text)
		m.textarea.MoveToEnd()
	case tea.PasteMsg:
		path := strings.ReplaceAll(msg.Content, "\\ ", " ")
		// try to get an image
		path, err := filepath.Abs(strings.TrimSpace(path))
		if err != nil {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		isAllowedType := false
		for _, ext := range filepicker.AllowedTypes {
			if strings.HasSuffix(path, ext) {
				isAllowedType = true
				break
			}
		}
		if !isAllowedType {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		tooBig, _ := filepicker.IsFileTooBig(path, filepicker.MaxAttachmentSize)
		if tooBig {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

		content, err := os.ReadFile(path)
		if err != nil {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		mimeBufferSize := min(512, len(content))
		mimeType := http.DetectContentType(content[:mimeBufferSize])
		fileName := filepath.Base(path)
		attachment := message.Attachment{FilePath: path, FileName: fileName, MimeType: mimeType, Content: content}
		return m, util.CmdHandler(filepicker.FilePickedMsg{
			Attachment: attachment,
		})

	case commands.ToggleYoloModeMsg:
		m.setEditorPrompt()
		return m, nil
	case tea.KeyPressMsg:
		cur := m.textarea.Cursor()
		curIdx := m.textarea.Width()*cur.Y + cur.X
		switch {
		// Open command palette when "/" is pressed on empty prompt
		case msg.String() == "/" && len(strings.TrimSpace(m.textarea.Value())) == 0:
			return m, util.CmdHandler(dialogs.OpenDialogMsg{
				Model: commands.NewCommandDialog(m.session.ID),
			})
		// Completions
		case msg.String() == "@" && !m.isCompletionsOpen &&
			// only show if beginning of prompt, or if previous char is a space or newline:
			(len(m.textarea.Value()) == 0 || unicode.IsSpace(rune(m.textarea.Value()[len(m.textarea.Value())-1]))):
			m.isCompletionsOpen = true
			m.currentQuery = ""
			m.completionsStartIndex = curIdx
			cmds = append(cmds, m.startCompletions)
		case m.isCompletionsOpen && curIdx <= m.completionsStartIndex:
			cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
		}
		if key.Matches(msg, DeleteKeyMaps.AttachmentDeleteMode) {
			m.deleteMode = true
			return m, nil
		}
		if key.Matches(msg, DeleteKeyMaps.DeleteAllAttachments) && m.deleteMode {
			m.deleteMode = false
			m.attachments = nil
			return m, nil
		}
		rune := msg.Code
		if m.deleteMode && unicode.IsDigit(rune) {
			num := int(rune - '0')
			m.deleteMode = false
			if num < 10 && len(m.attachments) > num {
				if num == 0 {
					m.attachments = m.attachments[num+1:]
				} else {
					m.attachments = slices.Delete(m.attachments, num, num+1)
				}
				return m, nil
			}
		}
		if key.Matches(msg, m.keyMap.OpenEditor) {
			if m.app.AgentCoordinator.IsSessionBusy(m.session.ID) {
				return m, util.ReportWarn("Agent is working, please wait...")
			}
			return m, m.openEditor(m.textarea.Value())
		}
		if key.Matches(msg, DeleteKeyMaps.Escape) {
			m.deleteMode = false
			return m, nil
		}
		if key.Matches(msg, m.keyMap.Newline) {
			m.textarea.InsertRune('\n')
			cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
		}
		// Handle Enter key
		if m.textarea.Focused() && key.Matches(msg, m.keyMap.SendMessage) {
			value := m.textarea.Value()
			if strings.HasSuffix(value, "\\") {
				// If the last character is a backslash, remove it and add a newline.
				m.textarea.SetValue(strings.TrimSuffix(value, "\\"))
			} else {
				// Otherwise, send the message
				return m, m.send()
			}
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	if m.textarea.Focused() {
		kp, ok := msg.(tea.KeyPressMsg)
		if ok {
			if kp.String() == "space" || m.textarea.Value() == "" {
				m.isCompletionsOpen = false
				m.currentQuery = ""
				m.completionsStartIndex = 0
				cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
			} else {
				word := m.textarea.Word()
				if strings.HasPrefix(word, "@") {
					// XXX: wont' work if editing in the middle of the field.
					m.completionsStartIndex = strings.LastIndex(m.textarea.Value(), word)
					m.currentQuery = word[1:]
					x, y := m.completionsPosition()
					x -= len(m.currentQuery)
					m.isCompletionsOpen = true
					cmds = append(cmds,
						util.CmdHandler(completions.FilterCompletionsMsg{
							Query:  m.currentQuery,
							Reopen: m.isCompletionsOpen,
							X:      x,
							Y:      y,
						}),
					)
				} else if m.isCompletionsOpen {
					m.isCompletionsOpen = false
					m.currentQuery = ""
					m.completionsStartIndex = 0
					cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *editorCmp) setEditorPrompt() {
	if m.app.Permissions.SkipRequests() {
		m.textarea.SetPromptFunc(4, yoloPromptFunc)
		return
	}
	m.textarea.SetPromptFunc(4, normalPromptFunc)
}

func (m *editorCmp) completionsPosition() (int, int) {
	cur := m.textarea.Cursor()
	if cur == nil {
		return m.x, m.y + 1 // adjust for padding
	}
	x := cur.X + m.x
	y := cur.Y + m.y + 1 // adjust for padding
	return x, y
}

func (m *editorCmp) Cursor() *tea.Cursor {
	cursor := m.textarea.Cursor()
	if cursor != nil {
		cursor.X = cursor.X + m.x + 1
		cursor.Y = cursor.Y + m.y + 1 // adjust for padding
	}
	return cursor
}

var readyPlaceholders = [...]string{
	"Awaiting Instructions..."
}

var workingPlaceholders = [...]string{
	"Cogitating...",
	"Calculating...",
	"Analyzing...",
	"Thinking...",
	"Synthesizing...",
	"Compiling thoughts...",
	"Crunching numbers...",
	"Spinning up neurons...",
	"Warming up the GPUs...",
	"Consulting the docs...",
	"Reading the tea leaves...",
	"Asking the magic 8-ball...",
	"Summoning inspiration...",
	"Channeling caffeine...",
	"Engaging brain cells...",
	"Connecting the dots...",
	"Weaving code magic...",
	"Brewing solutions...",
	"Baking fresh code...",
	"Cooking something up...",
	"Mixing ingredients...",
	"Following the recipe...",
	"Sharpening pencils...",
	"Stretching fingers...",
	"Limbering up...",
	"Doing mental push-ups...",
	"Exercising logic muscles...",
	"Flexing algorithms...",
	"Untangling spaghetti...",
	"Herding cats...",
	"Counting electrons...",
	"Aligning chakras...",
	"Consulting the elders...",
	"Deciphering runes...",
	"Spinning the hamster wheel...",
	"Feeding the hamsters...",
	"Waking up the hamsters...",
	"Charging flux capacitor...",
	"Reversing polarity...",
	"Reticulating splines...",
	"Generating witty banter...",
	"Contemplating existence...",
	"Having an existential crisis...",
	"Questioning everything...",
	"Finding meaning...",
	"Achieving enlightenment...",
	"Reaching nirvana...",
	"Transcending reality...",
	"Bending spoons...",
	"Warping spacetime...",
	"Folding proteins...",
	"Splitting atoms...",
	"Fusing neurons...",
	"Defragmenting brain...",
	"Clearing cache...",
	"Downloading inspiration...",
	"Uploading creativity...",
	"Synchronizing synapses...",
	"Optimizing pathways...",
	"Pruning decision trees...",
	"Watering logic gardens...",
	"Planting idea seeds...",
	"Harvesting thoughts...",
	"Mining for insights...",
	"Digging deeper...",
	"Excavating solutions...",
	"Unearthing answers...",
	"Polishing gems...",
	"Refining concepts...",
	"Distilling wisdom...",
	"Fermenting ideas...",
	"Aging like fine wine...",
	"Marinating in data...",
	"Simmering gently...",
	"Reducing complexity...",
	"Whisking vigorously...",
	"Kneading the dough...",
	"Letting it rise...",
	"Proofing concepts...",
	"Glazing the donut...",
	"Sprinkling magic dust...",
	"Adding secret sauce...",
	"Seasoning to taste...",
	"Garnishing output...",
	"Plating presentation...",
	"Serving hot...",
	"Bon appétit...",
}

func (m *editorCmp) randomizePlaceholders() {
	m.workingPlaceholder = workingPlaceholders[rand.Intn(len(workingPlaceholders))]
	m.readyPlaceholder = readyPlaceholders[rand.Intn(len(readyPlaceholders))]
}

// shimmerPlaceholder applies a sliding gradient shimmer effect to the placeholder text
func (m *editorCmp) shimmerPlaceholder(text string) string {
	if text == "" {
		return ""
	}
	
	t := styles.CurrentTheme()
	runes := []rune(text)
	var result strings.Builder
	
	// Create a sliding gradient effect across the text
	for i, r := range runes {
		// Calculate position in the shimmer wave (0.0 to 1.0)
		pos := float64(i) / float64(len(runes))
		
		// Create a wave that moves with shimmerOffset
		wave := pos - m.shimmerOffset
		if wave \u003c 0 {
			wave += 1.0
		}
		
		// Use a smooth gradient from muted to primary and back
		// Peak brightness at wave = 0.5
		brightness := 1.0 - 2.0*abs(wave-0.5)
		
		// Interpolate between muted and primary colors based on brightness
		var style lipgloss.Style
		if brightness \u003e 0.6 {
			// Bright part of shimmer - use primary/secondary gradient
			gradPos := (brightness - 0.6) / 0.4
			if int(m.shimmerOffset*10)%2 == 0 {
				style = t.S().Base.Foreground(t.Primary)
			} else {
				style = t.S().Base.Foreground(t.Secondary)
			}
		} else if brightness \u003e 0.3 {
			// Mid brightness - use blue
			style = t.S().Base.Foreground(t.Blue)
		} else {
			// Dim part - use muted color
			style = t.S().Muted
		}
		
		result.WriteString(style.Render(string(r)))
	}
	
	return result.String()
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x \u003c 0 {
		return -x
	}
	return x
}

func (m *editorCmp) View() string {
	t := styles.CurrentTheme()
	// Update placeholder with shimmer effect
	if m.app.AgentCoordinator != nil \u0026\u0026 m.app.AgentCoordinator.IsBusy() {
		m.textarea.Placeholder = m.shimmerPlaceholder(m.workingPlaceholder)
	} else {
		m.textarea.Placeholder = m.readyPlaceholder
	}
	if m.app.Permissions.SkipRequests() {
		m.textarea.Placeholder = "Yolo mode!"
	}
	if len(m.attachments) == 0 {
		content := t.S().Base.Padding(1).Render(
			m.textarea.View(),
		)
		return content
	}
	content := t.S().Base.Padding(0, 1, 1, 1).Render(
		lipgloss.JoinVertical(lipgloss.Top,
			m.attachmentsContent(),
			m.textarea.View(),
		),
	)
	return content
}

func (m *editorCmp) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 2)   // adjust for padding
	m.textarea.SetHeight(height - 2) // adjust for padding
	return nil
}

func (m *editorCmp) GetSize() (int, int) {
	return m.textarea.Width(), m.textarea.Height()
}

func (m *editorCmp) attachmentsContent() string {
	var styledAttachments []string
	t := styles.CurrentTheme()
	attachmentStyles := t.S().Base.
		MarginLeft(1).
		Background(t.FgMuted).
		Foreground(t.FgBase)
	for i, attachment := range m.attachments {
		var filename string
		if len(attachment.FileName) > 10 {
			filename = fmt.Sprintf(" %s %s...", styles.DocumentIcon, attachment.FileName[0:7])
		} else {
			filename = fmt.Sprintf(" %s %s", styles.DocumentIcon, attachment.FileName)
		}
		if m.deleteMode {
			filename = fmt.Sprintf("%d%s", i, filename)
		}
		styledAttachments = append(styledAttachments, attachmentStyles.Render(filename))
	}
	content := lipgloss.JoinHorizontal(lipgloss.Left, styledAttachments...)
	return content
}

func (m *editorCmp) SetPosition(x, y int) tea.Cmd {
	m.x = x
	m.y = y
	return nil
}

func (m *editorCmp) startCompletions() tea.Msg {
	ls := m.app.Config().Options.TUI.Completions
	depth, limit := ls.Limits()
	files, _, _ := fsext.ListDirectory(".", nil, depth, limit)
	slices.Sort(files)
	completionItems := make([]completions.Completion, 0, len(files))
	for _, file := range files {
		file = strings.TrimPrefix(file, "./")
		completionItems = append(completionItems, completions.Completion{
			Title: file,
			Value: FileCompletionItem{
				Path: file,
			},
		})
	}

	x, y := m.completionsPosition()
	return completions.OpenCompletionsMsg{
		Completions: completionItems,
		X:           x,
		Y:           y,
		MaxResults:  maxFileResults,
	}
}

// Blur implements Container.
func (c *editorCmp) Blur() tea.Cmd {
	c.textarea.Blur()
	return nil
}

// Focus implements Container.
func (c *editorCmp) Focus() tea.Cmd {
	return c.textarea.Focus()
}

// IsFocused implements Container.
func (c *editorCmp) IsFocused() bool {
	return c.textarea.Focused()
}

// Bindings implements Container.
func (c *editorCmp) Bindings() []key.Binding {
	return c.keyMap.KeyBindings()
}

// TODO: most likely we do not need to have the session here
// we need to move some functionality to the page level
func (c *editorCmp) SetSession(session session.Session) tea.Cmd {
	c.session = session
	return nil
}

func (c *editorCmp) IsCompletionsOpen() bool {
	return c.isCompletionsOpen
}

func (c *editorCmp) HasAttachments() bool {
	return len(c.attachments) > 0
}

func normalPromptFunc(info textarea.PromptInfo) string {
	t := styles.CurrentTheme()
	if info.LineNumber == 0 {
		return "  > "
	}
	if info.Focused {
		return t.S().Base.Foreground(t.GreenDark).Render("::: ")
	}
	return t.S().Muted.Render("::: ")
}

func yoloPromptFunc(info textarea.PromptInfo) string {
	t := styles.CurrentTheme()
	if info.LineNumber == 0 {
		if info.Focused {
			return fmt.Sprintf("%s ", t.YoloIconFocused)
		} else {
			return fmt.Sprintf("%s ", t.YoloIconBlurred)
		}
	}
	if info.Focused {
		return fmt.Sprintf("%s ", t.YoloDotsFocused)
	}
	return fmt.Sprintf("%s ", t.YoloDotsBlurred)
}

func New(app *app.App) Editor {
	t := styles.CurrentTheme()
	ta := textarea.New()
	ta.SetStyles(t.S().TextArea)
	ta.ShowLineNumbers = false
	ta.CharLimit = -1
	ta.SetVirtualCursor(false)
	ta.Focus()
	e := &editorCmp{
		// TODO: remove the app instance from here
		app:      app,
		textarea: ta,
		keyMap:   DefaultEditorKeyMap(),
	}
	e.setEditorPrompt()

	e.randomizePlaceholders()
	e.textarea.Placeholder = e.readyPlaceholder

	return e
}

