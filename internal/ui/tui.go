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

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msaeedsaeedi/again/internal/domain"
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

type Model struct {
	cfg        *domain.RunConfig
	started    int
	completed  int
	lastResult *domain.RunResult
	finished   bool
	quit       bool

	spinner   spinner.Model
	progress  progress.Model
	width     int
	height    int
	logLines  []string
	maxLines  int
	currentID int
	mu        sync.Mutex
}

func (m *Model) Init() tea.Cmd { return m.spinner.Tick }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startMsg:
		m.mu.Lock()
		m.started++
		m.currentID = msg.runID
		m.mu.Unlock()
	case completeMsg:
		m.mu.Lock()
		m.completed++
		m.lastResult = &msg.result
		m.mu.Unlock()
	case streamMsg:
		m.appendLog(msg)
	case allCompleteMsg:
		m.finished = true
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quit = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 10
	}

	return m, nil
}

// TODO: Clean up the UI
func (m *Model) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.width == 0 {
		return "starting…"
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	sectionStyle := lipgloss.NewStyle().Padding(1, 2)
	logStyle := lipgloss.NewStyle().Width(m.width-4).MaxWidth(m.width-4).Padding(0, 2)

	var b strings.Builder

	b.WriteString(sectionStyle.Render(headerStyle.Render("again")))
	b.WriteString("\n")

	status := "running"
	if m.finished {
		status = "completed"
	}

	b.WriteString(sectionStyle.Render(fmt.Sprintf("%s  command: %s", m.spinner.View(), strings.Join(m.cfg.Command, " "))))
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render(fmt.Sprintf("runs: %d/%d  status: %s", m.completed, m.cfg.Times, status)))
	b.WriteString("\n")

	progress := m.progress.ViewAs(progressPercent(m.completed, m.cfg.Times))
	b.WriteString(sectionStyle.Render(progress))
	b.WriteString("\n")

	if m.lastResult != nil {
		label := "ok"
		if !m.lastResult.Success {
			label = fmt.Sprintf("failed (exit %d)", m.lastResult.ExitCode)
		}
		fmt.Fprintf(&b, "%s\n", sectionStyle.Render(fmt.Sprintf("last: #%d %s in %v", m.lastResult.ID, label, m.lastResult.Duration.Round(time.Millisecond))))
	}

	if len(m.logLines) > 0 {
		b.WriteString(sectionStyle.Render(headerStyle.Render("stream")))
		b.WriteString("\n")
		b.WriteString(logStyle.Render(strings.Join(m.logLines, "\n")))
		b.WriteString("\n")
	}

	if m.finished {
		b.WriteString(sectionStyle.Render("done. press q to exit"))
	} else {
		b.WriteString(sectionStyle.Render("press q or ctrl+c to stop"))
	}

	return b.String()
}

func NewModel(cfg *domain.RunConfig) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithSolidFill("#00db9a"))
	p.Full = '─'

	return &Model{
		cfg:      cfg,
		spinner:  s,
		progress: p,
		maxLines: 20,
		logLines: make([]string, 0, 20),
	}
}

func NewTUIFormatter(cfg *domain.RunConfig) *TUIFormatter {
	return &TUIFormatter{model: NewModel(cfg), ready: make(chan struct{})}
}

func (f *TUIFormatter) Run(ctx context.Context) error {
	options := []tea.ProgramOption{tea.WithAltScreen()}
	if ctx != nil {
		options = append(options, tea.WithContext(ctx))
	}

	f.program = tea.NewProgram(f.model, options...)
	f.once.Do(func() { close(f.ready) })

	_, err := f.program.Run()
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, tea.ErrProgramKilled) {
		return nil
	}
	return err
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
	return &tuiWriter{program: f.program, isErr: false, formatter: f, runID: id}, &tuiWriter{program: f.program, isErr: true, formatter: f, runID: id}
}

func (w *tuiWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		// If program not yet ready, try to read from formatter (it will be set shortly after Run starts)
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

// TODO: Update the logging mechanism
func (m *Model) appendLog(msg streamMsg) {
	m.mu.Lock()
	defer m.mu.Unlock()

	line := msg.text
	if msg.runID > 0 {
		line = fmt.Sprintf("run %d %s", msg.runID, strings.TrimRight(msg.text, "\n"))
	}
	if msg.isErr {
		line = fmt.Sprintf("stderr: %s", strings.TrimRight(line, "\n"))
	} else {
		line = fmt.Sprintf("stdout: %s", strings.TrimRight(line, "\n"))
	}

	m.logLines = append(m.logLines, line)
	if len(m.logLines) > m.maxLines {
		m.logLines = m.logLines[len(m.logLines)-m.maxLines:]
	}
}

func progressPercent(completed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(completed) / float64(total)
}
