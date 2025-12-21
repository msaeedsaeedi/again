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
		m.finished = true
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
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *Model) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.width == 0 {
		return "Initializing..."
	}

	// Layout Dimensions
	// Account for margins (1 line top/bottom, 2 cols left/right = 4 total width, 2 total height)
	availWidth := max(20, m.width-4)
	availHeight := max(10, m.height-2)

	// Use roughly 25% for sidebar, but keep min/max bounds
	sidebarW := max(30, availWidth/4)
	mainW := availWidth - sidebarW - 1 // -1 for potential gutter

	footerHeight := 1 // 1 line of text
	contentH := max(10, availHeight-footerHeight)

	// --- 1. Sidebar Construction ---
	var sb strings.Builder

	sb.WriteString(styleBoldWhite.Render("RUN HISTORY"))
	sb.WriteString("\n\n")

	// Calculate visible range for sidebar with scrolling support
	// Reserve space for header (3 lines: title + blank line + potential scroll indicator)
	sidebarVisibleLines := contentH - 4

	// Auto-scroll sidebar when selected run moves down
	if m.selectedRun >= m.sidebarScrollOffset+sidebarVisibleLines {
		m.sidebarScrollOffset = m.selectedRun - sidebarVisibleLines + 1
	}

	// Calculate visible window
	startIdx := m.sidebarScrollOffset
	endIdx := min(len(m.runs), startIdx+sidebarVisibleLines)

	// Show scroll indicators (without extra newline to avoid shifting)
	if startIdx > 0 {
		sb.WriteString(styleDim.Render("  ▲ more above"))
		sb.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		run := m.runs[i]

		// Icon & Color Logic
		var icon, statusStr string
		var runStyle lipgloss.Style

		switch run.status {
		case "success":
			icon = "✓"
			statusStr = "0"
			runStyle = styleSuccess
		case "failed":
			icon = "✗"
			statusStr = fmt.Sprintf("%d", run.exitCode)
			runStyle = styleFailure
		case "running":
			icon = "..."
			statusStr = ""
			runStyle = styleRunning
		default: // pending
			icon = "-"
			statusStr = ""
			runStyle = stylePending
		}

		timeStr := ""
		if !run.startedAt.IsZero() {
			timeStr = run.startedAt.Format("15:04:05")
		}

		rowLeft := fmt.Sprintf("Run #%03d %s %-2s", run.id, icon, statusStr)

		var line string
		if i == m.selectedRun {
			// Active Selection: Add "┃" prefix and blue color
			if timeStr != "" {
				line = fmt.Sprintf("┃ %-18s %s", rowLeft, timeStr)
			} else {
				line = fmt.Sprintf("┃ %s", rowLeft)
			}
			sb.WriteString(styleActive.Render(line))
		} else {
			// Inactive: Padding instead of bar
			if timeStr != "" {
				line = fmt.Sprintf("  %-18s %s", rowLeft, timeStr)
			} else {
				line = fmt.Sprintf("  %s", rowLeft)
			}
			sb.WriteString(runStyle.Render(line))
		}
		sb.WriteString("\n")
	}

	// Show scroll indicator if there are more runs below
	if endIdx < len(m.runs) {
		sb.WriteString(styleDim.Render("  ▼ more below"))
	}

	leftPanel := styleSidebar.Width(sidebarW).MaxWidth(sidebarW).Height(contentH).Render(sb.String())

	// --- 2. Main Area Construction ---
	var main strings.Builder

	if m.selectedRun < len(m.runs) {
		run := m.runs[m.selectedRun]

		// Header Section
		main.WriteString(styleBoldWhite.Render(fmt.Sprintf("RUN DETAILS: #%03d", run.id)))
		main.WriteString("\n\n")

		// Metadata Grid
		main.WriteString(styleBoldWhite.Render("Command"))
		fmt.Fprintf(&main, "\n  > %s\n\n", strings.Join(m.cfg.Command, " "))
		main.WriteString(styleBoldWhite.Render("Status") + "\n")

		statText := "Pending"
		switch run.status {
		case "success":
			statText = styleSuccess.Render("Success (Exit Code: 0)")
		case "failed":
			statText = styleFailure.Render(fmt.Sprintf("Failed (Exit Code: %d)", run.exitCode))
		case "running":
			statText = styleRunning.Render("Running...")
		}
		main.WriteString("  " + statText + "\n\n")

		if run.duration != 0 || run.status == "running" {
			dur := run.duration
			if run.status == "running" {
				if !m.lastTickTime.IsZero() {
					dur = m.lastTickTime.Sub(run.startedAt)
				} else {
					dur = time.Since(run.startedAt)
				}
			}
			main.WriteString(styleBoldWhite.Render("Duration") + "\n")
			main.WriteString(fmt.Sprintf("  %s\n\n", dur.Round(time.Millisecond)))
		}

		// LOGS Section
		main.WriteString(styleBoldWhite.Render("OUTPUT LOGS"))
		main.WriteString("\n")

		runLogEntries := m.runLogs[run.id]
		var runLogs []string
		for _, entry := range runLogEntries {
			runLogs = append(runLogs, entry.text)
		}

		// Scroll Logic
		// Approximating header lines usage: ~12-14 lines
		logAreaHeight := max(5, contentH-14)

		totalLogLines := len(runLogs)

		// Auto-scroll to end if enabled
		if m.autoScroll && totalLogLines > logAreaHeight {
			m.scrollOffset = totalLogLines - logAreaHeight
		}

		start := m.scrollOffset

		if start > totalLogLines-logAreaHeight {
			start = max(0, totalLogLines-logAreaHeight)
		}
		if start < 0 {
			start = 0
		}
		end := min(totalLogLines, start+logAreaHeight)

		for i := start; i < end; i++ {
			main.WriteString(runLogs[i] + "\n")
		}

		if end < totalLogLines {
			main.WriteString(styleDim.Render("... (scroll down for more) ..."))
		}
	}

	rightPanel := styleMain.Width(mainW).Height(contentH).Render(main.String())

	// --- 3. Footer Construction ---

	// Left side: Progress Section
	progressStr := fmt.Sprintf("%d/%d", m.completed, m.cfg.Times)
	stateStr := "Active"
	if m.finished {
		stateStr = "Complete"
	}
	leftSection := styleHelpText.Render(progressStr + " " + stateStr)

	// Right side: Help Section with proper spacing
	var helpItems []string

	// Navigation
	helpItems = append(helpItems, styleHelpKey.Render("↑/k")+styleHelpText.Render(" navigate"))
	helpItems = append(helpItems, styleHelpKey.Render("pgup/pgdn")+styleHelpText.Render(" scroll"))

	// Quit
	helpItems = append(helpItems, styleHelpKey.Render("q")+styleHelpText.Render(" quit"))

	// Join help items
	rightSection := strings.Join(helpItems, "   ")

	// Calculate spacing to spread items across footer
	leftWidth := lipgloss.Width(leftSection)
	rightWidth := lipgloss.Width(rightSection)
	spacerWidth := max(2, availWidth-leftWidth-rightWidth-4) // -4 for padding
	spacer := strings.Repeat(" ", spacerWidth)

	footerLine := leftSection + spacer + rightSection

	footerPanel := styleFooter.Width(availWidth).Render(footerLine)

	// --- 4. Final Composition ---
	// Horizontal join sidebar + main
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Vertical join body + footer
	screen := lipgloss.JoinVertical(lipgloss.Left, body, footerPanel)

	// Apply margin to entire screen
	return styleScreen.Render(screen)
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
