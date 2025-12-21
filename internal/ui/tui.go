package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msaeedsaeedi/again/internal/domain"
)

var (
	// Colors (Mental Model)
	colorActiveBlue = lipgloss.Color("39")  // Bright Cyan/Blue for active elements
	colorDimGray    = lipgloss.Color("240") // Faded text for timestamps, logs
	colorGreen      = lipgloss.Color("42")  // Success
	colorRed        = lipgloss.Color("196") // Failure
	colorYellow     = lipgloss.Color("220") // Running/Pending
	colorWhite      = lipgloss.Color("255")
	colorLightGray  = lipgloss.Color("250") // Slightly brighter gray for keys

	// Text Styles
	styleBoldWhite = lipgloss.NewStyle().Bold(true).Foreground(colorWhite)
	styleDim       = lipgloss.NewStyle().Foreground(colorDimGray)
	styleActive    = lipgloss.NewStyle().Foreground(colorActiveBlue).Bold(true)
	styleSuccess   = lipgloss.NewStyle().Foreground(colorGreen)
	styleFailure   = lipgloss.NewStyle().Foreground(colorRed)
	stylePending   = lipgloss.NewStyle().Foreground(colorDimGray)
	styleRunning   = lipgloss.NewStyle().Foreground(colorYellow)

	// Footer Styles
	styleHelpKey  = lipgloss.NewStyle().Foreground(colorLightGray)
	styleHelpText = lipgloss.NewStyle().Foreground(colorDimGray)

	// Layout Styles (Negative Space)
	styleSidebar = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
	styleMain    = lipgloss.NewStyle().PaddingLeft(4)
	styleFooter  = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(1).PaddingBottom(1)
	styleScreen  = lipgloss.NewStyle().Margin(1, 2)
)

type TUIFormatter struct {
	model   *Model
	program *tea.Program
	runID   int64
	ready   chan struct{}
	once    sync.Once
}

type startMsg struct{ runID int }
type completeMsg struct{ result domain.RunResult }
type allCompleteMsg struct{}
type tickMsg time.Time

type streamMsg struct {
	text  string
	isErr bool
	runID int
}

type tuiWriter struct {
	program   *tea.Program
	isErr     bool
	formatter *TUIFormatter
	runID     int
}

type runState struct {
	id         int
	status     string // "pending", "running", "success", "failed"
	exitCode   int
	duration   time.Duration
	startedAt  time.Time
	finishedAt time.Time
}

type logLine struct {
	timestamp time.Time
	text      string // Pre-styled text with timestamp
	isErr     bool
}

type Model struct {
	cfg                 *domain.RunConfig
	started             int
	completed           int
	finished            bool
	quit                bool
	width               int
	height              int
	runs                []runState
	selectedRun         int
	runLogs             map[int][]logLine // Per-run log storage
	maxLinesPerRun      int               // Max lines per individual run
	scrollOffset        int               // Vertical scroll for log view
	sidebarScrollOffset int               // Vertical scroll for sidebar
	autoScroll          bool              // Auto-scroll to latest logs
	lastTickTime        time.Time         // Last tick time for consistent duration calculation
	mu                  sync.Mutex
}

func NewModel(cfg *domain.RunConfig) *Model {
	runs := make([]runState, cfg.Times)
	for i := 0; i < cfg.Times; i++ {
		runs[i] = runState{
			id:     i + 1,
			status: "pending",
		}
	}

	return &Model{
		cfg:            cfg,
		runLogs:        make(map[int][]logLine),
		maxLinesPerRun: 10000,
		runs:           runs,
		selectedRun:    0,
		autoScroll:     true,
	}
}

func NewTUIFormatter(cfg *domain.RunConfig) *TUIFormatter {
	return &TUIFormatter{model: NewModel(cfg), ready: make(chan struct{})}
}

func (m *Model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startMsg:
		m.mu.Lock()
		m.started++
		for i := range m.runs {
			if m.runs[i].id == msg.runID {
				m.runs[i].status = "running"
				m.runs[i].startedAt = time.Now()
				break
			}
		}
		m.mu.Unlock()

	case completeMsg:
		m.mu.Lock()
		m.completed++
		for i := range m.runs {
			if m.runs[i].id == msg.result.ID {
				if msg.result.Success {
					m.runs[i].status = "success"
				} else {
					m.runs[i].status = "failed"
				}
				m.runs[i].exitCode = msg.result.ExitCode
				m.runs[i].duration = msg.result.Duration
				m.runs[i].finishedAt = msg.result.FinishedAt
				break
			}
		}
		m.mu.Unlock()

	case streamMsg:
		m.appendLog(msg)
		return m, nil

	case tickMsg:
		m.mu.Lock()
		m.lastTickTime = time.Time(msg)
		hasActiveRuns := !m.finished
		m.mu.Unlock()

		if hasActiveRuns {
			return m, tick()
		}
		return m, nil

	case allCompleteMsg:
		m.mu.Lock()
		m.finished = true
		m.mu.Unlock()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			m.mu.Lock()
			if m.selectedRun > 0 {
				m.selectedRun--
				m.autoScroll = true
				if m.selectedRun < m.sidebarScrollOffset {
					m.sidebarScrollOffset = m.selectedRun
				}
			}
			m.mu.Unlock()
		case "down", "j":
			m.mu.Lock()
			if m.selectedRun < len(m.runs)-1 {
				m.selectedRun++
				m.autoScroll = true
			}
			m.mu.Unlock()
		case "home":
			m.mu.Lock()
			m.scrollOffset = 0
			m.autoScroll = false
			m.mu.Unlock()
		case "end":
			m.mu.Lock()
			m.scrollOffset = 999999
			m.autoScroll = true
			m.mu.Unlock()
		case "pgup":
			m.mu.Lock()
			m.scrollOffset = max(0, m.scrollOffset-10)
			m.autoScroll = false
			m.mu.Unlock()
		case "pgdown":
			m.mu.Lock()
			m.scrollOffset += 10
			m.autoScroll = false
			m.mu.Unlock()
		}

	case tea.WindowSizeMsg:
		m.mu.Lock()
		m.width = msg.Width
		m.height = msg.Height
		m.mu.Unlock()
	}

	return m, nil
}

func (m *Model) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.width == 0 {
		return "Initializing..."
	}

	availWidth := max(20, m.width-4)
	availHeight := max(10, m.height-2)
	sidebarW := max(30, availWidth/4)
	mainW := availWidth - sidebarW - 1
	footerHeight := 3
	contentH := max(10, availHeight-footerHeight)

	sidebar := m.renderSidebar(sidebarW, contentH)
	mainPanel := m.renderMainPanel(mainW, contentH)
	footer := m.renderFooter(availWidth)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainPanel)
	screen := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	return styleScreen.Render(screen)
}

func (m *Model) renderSidebar(width, height int) string {
	var sb strings.Builder

	sb.WriteString(styleBoldWhite.Render("RUN HISTORY"))
	sb.WriteString("\n\n")

	visibleLines := height - 4

	if m.selectedRun >= m.sidebarScrollOffset+visibleLines {
		m.sidebarScrollOffset = m.selectedRun - visibleLines + 1
	}

	startIdx := m.sidebarScrollOffset
	endIdx := min(len(m.runs), startIdx+visibleLines)

	if startIdx > 0 {
		sb.WriteString(styleDim.Render("  ▲ more above"))
		sb.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		sb.WriteString(m.renderRunLine(i))
		sb.WriteString("\n")
	}

	if endIdx < len(m.runs) {
		sb.WriteString(styleDim.Render("  ▼ more below"))
	}

	return styleSidebar.Width(width).MaxWidth(width).Height(height).Render(sb.String())
}

func (m *Model) renderRunLine(index int) string {
	run := m.runs[index]

	icon, statusStr, runStyle := m.getRunStatusDisplay(run)

	timeStr := ""
	if !run.startedAt.IsZero() {
		timeStr = run.startedAt.Format("15:04:05")
	}

	rowLeft := fmt.Sprintf("Run #%03d %s %-2s", run.id, icon, statusStr)

	var line string
	if index == m.selectedRun {
		if timeStr != "" {
			line = fmt.Sprintf("┃ %-18s %s", rowLeft, timeStr)
		} else {
			line = fmt.Sprintf("┃ %s", rowLeft)
		}
		return styleActive.Render(line)
	}

	if timeStr != "" {
		line = fmt.Sprintf("  %-18s %s", rowLeft, timeStr)
	} else {
		line = fmt.Sprintf("  %s", rowLeft)
	}
	return runStyle.Render(line)
}

func (m *Model) getRunStatusDisplay(run runState) (icon, statusStr string, style lipgloss.Style) {
	switch run.status {
	case "success":
		return "✓", "0", styleSuccess
	case "failed":
		return "✗", fmt.Sprintf("%d", run.exitCode), styleFailure
	case "running":
		return "...", "", styleRunning
	default:
		return "-", "", stylePending
	}
}

func (m *Model) renderMainPanel(width, height int) string {
	var main strings.Builder

	if m.selectedRun >= len(m.runs) {
		return styleMain.Width(width).Render("")
	}

	run := m.runs[m.selectedRun]

	main.WriteString(styleBoldWhite.Render(fmt.Sprintf("RUN DETAILS: #%03d", run.id)))
	main.WriteString("\n\n")

	m.renderCommandSection(&main)
	m.renderStatusSection(&main, run)
	m.renderDurationSection(&main, run)
	m.renderLogsSection(&main, run, height)

	return styleMain.Width(width).Height(height).Render(main.String())
}

func (m *Model) renderCommandSection(w *strings.Builder) {
	w.WriteString(styleBoldWhite.Render("Command"))
	fmt.Fprintf(w, "\n  > %s\n\n", strings.Join(m.cfg.Command, " "))
}

func (m *Model) renderStatusSection(w *strings.Builder, run runState) {
	w.WriteString(styleBoldWhite.Render("Status") + "\n")

	var statText string
	switch run.status {
	case "success":
		statText = styleSuccess.Render("Success (Exit Code: 0)")
	case "failed":
		statText = styleFailure.Render(fmt.Sprintf("Failed (Exit Code: %d)", run.exitCode))
	case "running":
		statText = styleRunning.Render("Running...")
	default:
		statText = "Pending"
	}

	w.WriteString("  " + statText + "\n\n")
}

func (m *Model) renderDurationSection(w *strings.Builder, run runState) {
	// Always render Duration section to maintain consistent height
	dur := time.Duration(0)

	if run.duration != 0 {
		dur = run.duration
	} else if run.status == "running" {
		if !m.lastTickTime.IsZero() {
			dur = m.lastTickTime.Sub(run.startedAt)
		} else {
			dur = time.Since(run.startedAt)
		}
	}

	w.WriteString(styleBoldWhite.Render("Duration") + "\n")
	if dur > 0 {
		w.WriteString(fmt.Sprintf("  %s\n\n", dur.Round(time.Millisecond)))
	} else {
		w.WriteString("  -\n\n")
	}
}

func (m *Model) renderLogsSection(w *strings.Builder, run runState, contentHeight int) {
	w.WriteString(styleBoldWhite.Render("OUTPUT LOGS"))
	w.WriteString("\n")

	runLogEntries := m.runLogs[run.id]
	var runLogs []string
	for _, entry := range runLogEntries {
		runLogs = append(runLogs, entry.text)
	}

	// Reserve lines: title(1) + blank(1) + header sections(~12) + scroll indicator(1)
	logAreaHeight := max(5, contentHeight-15)
	totalLogLines := len(runLogs)

	if m.autoScroll && totalLogLines > logAreaHeight {
		m.scrollOffset = totalLogLines - logAreaHeight
	}

	start := max(0, min(m.scrollOffset, totalLogLines-logAreaHeight))
	end := min(totalLogLines, start+logAreaHeight)

	// Always render exactly logAreaHeight lines to prevent layout shift
	linesRendered := 0
	for i := start; i < end; i++ {
		w.WriteString(runLogs[i] + "\n")
		linesRendered++
	}

	// Fill remaining space with empty lines to maintain consistent height
	for linesRendered < logAreaHeight {
		w.WriteString("\n")
		linesRendered++
	}

	// Scroll indicator always rendered (consistent height)
	if end < totalLogLines {
		w.WriteString(styleDim.Render("... (scroll down for more) ..."))
	} else {
		w.WriteString(" ") // Placeholder to maintain height
	}
}

func (m *Model) renderFooter(width int) string {
	progressStr := fmt.Sprintf("%d/%d", m.completed, m.cfg.Times)
	stateStr := "Active"
	if m.finished {
		stateStr = "Complete"
	}
	leftSection := styleHelpText.Render(progressStr + " " + stateStr)

	var helpItems []string
	helpItems = append(helpItems, styleHelpKey.Render("↑/k")+styleHelpText.Render(" navigate"))
	helpItems = append(helpItems, styleHelpKey.Render("pgup/pgdn")+styleHelpText.Render(" scroll"))
	helpItems = append(helpItems, styleHelpKey.Render("q")+styleHelpText.Render(" quit"))

	rightSection := strings.Join(helpItems, "   ")

	leftWidth := lipgloss.Width(leftSection)
	rightWidth := lipgloss.Width(rightSection)
	spacerWidth := max(2, width-leftWidth-rightWidth-4)
	spacer := strings.Repeat(" ", spacerWidth)

	footerLine := leftSection + spacer + rightSection

	return styleFooter.Width(width).Render(footerLine)
}

func (m *Model) appendLog(msg streamMsg) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := strings.Split(strings.TrimRight(msg.text, "\n"), "\n")
	timestamp := time.Now()

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Visual timestamp
		ts := styleDim.Render("[" + timestamp.Format("15:04:05") + "] ")

		var styledLine string
		if msg.isErr {
			styledLine = styleFailure.Render(line)
		} else {
			styledLine = styleDim.Render(line) // Normal output
		}

		// Create log entry
		entry := logLine{
			timestamp: timestamp,
			text:      ts + styledLine,
			isErr:     msg.isErr,
		}

		// Append to run-specific logs
		m.runLogs[msg.runID] = append(m.runLogs[msg.runID], entry)

		// Prune old logs for this run to prevent memory bloat
		if len(m.runLogs[msg.runID]) > m.maxLinesPerRun {
			m.runLogs[msg.runID] = m.runLogs[msg.runID][len(m.runLogs[msg.runID])-m.maxLinesPerRun:]
		}
	}
}

func (f *TUIFormatter) Run(ctx context.Context) error {
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if ctx != nil {
		opts = append(opts, tea.WithContext(ctx))
	}

	f.program = tea.NewProgram(f.model, opts...)
	f.once.Do(func() { close(f.ready) })

	_, err := f.program.Run()
	if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return err
	}
	return nil
}

func (f *TUIFormatter) WaitReady(ctx context.Context) error {
	select {
	case <-f.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *TUIFormatter) OnStart(runID int) {
	atomic.StoreInt64(&f.runID, int64(runID))
	if f.program != nil {
		f.program.Send(startMsg{runID: runID})
	}
}

func (f *TUIFormatter) OnComplete(result domain.RunResult) {
	if f.program != nil {
		f.program.Send(completeMsg{result: result})
	}
}

func (f *TUIFormatter) OnFinish() {
	if f.program != nil {
		f.program.Send(allCompleteMsg{})
	}
}

func (f *TUIFormatter) GetOutputWriters() (stdout, stderr io.Writer) {
	id := int(atomic.LoadInt64(&f.runID))
	return &tuiWriter{program: f.program, isErr: false, formatter: f, runID: id},
		&tuiWriter{program: f.program, isErr: true, formatter: f, runID: id}
}

func (w *tuiWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		prog := w.program
		if prog == nil {
			prog = w.formatter.program
		}
		if prog != nil {
			prog.Send(streamMsg{text: string(p), isErr: w.isErr, runID: w.runID})
		}
	}
	return len(p), nil
}
