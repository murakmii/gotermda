package shell

import (
	"os"
	"os/exec"
	"syscall"
)

type Shell struct {
	cmd *exec.Cmd
}

func Start(path string, slave *os.File) (*Shell, error) {
	sh := &Shell{cmd: exec.Command(path)}
	sh.cmd.Stdout = slave
	sh.cmd.Stdin = slave
	sh.cmd.Stderr = slave
	sh.cmd.Env = []string{
		"TERM=xterm",
	}
	sh.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}

	if err := sh.cmd.Start(); err != nil {
		return nil, err
	}

	return sh, nil
}

func (sh *Shell) Pid() int {
	return sh.cmd.Process.Pid
}

func (sh *Shell) Wait() error {
	return sh.cmd.Wait()
}
