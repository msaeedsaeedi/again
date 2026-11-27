//go:build windows

package infra

import (
	"os/exec"
	"strings"
	"unsafe"

	"github.com/msaeedsaeedi/again/internal/domain"
	"golang.org/x/sys/windows"
)

type processGuard struct {
	job windows.Handle
}

func (g *processGuard) Attach(cmd *exec.Cmd) {
	if g == nil || cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return
	}

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}

	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return
	}

	proc, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return
	}
	defer func() { _ = windows.CloseHandle(proc) }()

	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		_ = windows.CloseHandle(job)
		return
	}

	g.job = job
}

func (g *processGuard) Kill(cmd *exec.Cmd) {
	if g != nil && g.job != 0 {
		_ = windows.TerminateJobObject(g.job, 1)
		return
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (g *processGuard) Close() {
	if g != nil && g.job != 0 {
		_ = windows.CloseHandle(g.job)
		g.job = 0
	}
}

func setupProcessGroup(cmd *exec.Cmd) *processGuard {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &windows.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW
	return &processGuard{}
}

func createCmd(cfg *domain.RunConfig) *exec.Cmd {
	var cmd *exec.Cmd
	if len(cfg.Command) == 1 {
		cmdStr := strings.ReplaceAll(cfg.Command[0], "/", "\\")
		cmd = exec.Command("cmd", "/C", cmdStr)

		if needsCmdQuoting(cmdStr) {
			cmd.SysProcAttr = &windows.SysProcAttr{
				CmdLine: `cmd /C "` + escapeCmdArg(cmdStr) + `"`,
			}
		}
	} else {
		args := make([]string, len(cfg.Command[1:]))
		for i, arg := range cfg.Command[1:] {
			args[i] = strings.ReplaceAll(arg, "/", "\\")
		}
		cmd = exec.Command(cfg.Command[0], args...)
	}

	return cmd
}

func needsCmdQuoting(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '&', '|', '<', '>', '^', '(', ')':
			return true
		}
	}
	return false
}

func escapeCmdArg(s string) string {
	// Escapa comillas dobles para que cmd.exe no cierre prematuramente la cadena
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			out = append(out, '\\', '"')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}
