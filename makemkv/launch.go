package makemkv

import (
	"context"
	"os/exec"
	"slices"
)

var (
	defaultArgs = []string{
		// Discs are fingerprinted based on detected
		// playlists. Therefore, the minimum length for an
		// acceptable playlist must be consistent across
		// invocations. 120 seconds is the `makemkvcon`
		// default, but it's user-customizeable. Set it back
		// to the default explicitly.
		//
		// Ideally, this could be set to 0, but that would
		// substantially increase processing time for discs
		// tha thave a lot of short playlists, as `makemkvcon`
		// tries to access each one and process some metadata.
		"--minlength=120",
		// Enable "robot-mode" output. This causes
		// `makemkvcon` to produce messages in a parseable
		// format, which this tool needs to be able to do its
		// job.
		"--robot",
		// Since `makemkvcon` is being run as a subprocess,
		// all messages can go to stdout to be captured.
		"--messages=-stdout",
		// Send progress messages to the same location as
		// messages so they can also be captured.
		"--progress=-same",
	}
)

type MakeMkvProcess struct {
	cmd *exec.Cmd
	// Args holds the arguments the process was launched with,
	// including any arguments that were added by default.
	Args []string
}

func NewProcess(ctx context.Context, makemkvcon string, args []string) (*MakeMkvProcess, error) {
	result := &MakeMkvProcess{
		Args:  slices.Concat(defaultArgs, args),
	}
	result.cmd = exec.CommandContext(ctx, makemkvcon, result.Args...)
	return result, nil
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
