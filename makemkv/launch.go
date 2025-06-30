package makemkv

import (
	"os/exec"
	"slices"
)

var (
	defaultArgs = []string{"--robot", "--messages=-stdout", "--progress=-same"}
)

type MakeMkvProcess struct {
	cmd *exec.Cmd
}

func NewProcess(makemkvcon string, args []string) (*MakeMkvProcess, error) {
	cmd := exec.Command(makemkvcon, slices.Concat(defaultArgs, args)...)
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
