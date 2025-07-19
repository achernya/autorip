package makemkv

import (
	"os/exec"
	"testing"
)

func TestLaunch(t *testing.T) {
	path, err := exec.LookPath("echo")
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewProcess(t.Context(), path, []string{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Wait() //nolint:errcheck
	p.Kill()
}
