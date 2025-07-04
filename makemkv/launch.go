package makemkv

import (
	"context"
	"os/exec"
	"slices"
)

var (
	defaultArgs = []string{"--robot", "--messages=-stdout", "--progress=-same"}
)

type MakeMkvProcess struct {
	cmd *exec.Cmd
}

func NewProcess(ctx context.Context, makemkvcon string, args []string) (*MakeMkvProcess, error) {
	cmd := exec.CommandContext(ctx, makemkvcon, slices.Concat(defaultArgs, args)...)
	return &MakeMkvProcess{cmd: cmd}, nil
}

func (m *MakeMkvProcess) Start() (*MakeMkvParser, error) {
	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := m.cmd.Start(); err != nil {
		return nil, err
	}
	return NewParser(stdout), nil
}

func (m *MakeMkvProcess) Wait() error {
	return m.cmd.Wait()
}

func (m *MakeMkvProcess) Kill() {
	m.cmd.Process.Kill()
}
