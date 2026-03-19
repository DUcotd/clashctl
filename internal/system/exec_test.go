package system

import (
	"errors"
	"testing"
)

type fakeRunner struct {
	output string
	err    error
	runErr error
	calls  []runnerCall
}

type runnerCall struct {
	name string
	args []string
}

func (f *fakeRunner) CombinedOutput(name string, args ...string) (string, error) {
	f.calls = append(f.calls, runnerCall{name: name, args: append([]string(nil), args...)})
	return f.output, f.err
}

func (f *fakeRunner) Run(name string, args ...string) error {
	f.calls = append(f.calls, runnerCall{name: name, args: append([]string(nil), args...)})
	return f.runErr
}

func TestRunCommandUsesInjectedRunner(t *testing.T) {
	runner := &fakeRunner{output: "ok"}
	restore := SetCommandRunner(runner)
	defer restore()

	got, err := RunCommand("systemctl", "status", "mihomo")
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("RunCommand() = %q, want %q", got, "ok")
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner call count = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "systemctl" {
		t.Fatalf("runner call name = %q, want %q", runner.calls[0].name, "systemctl")
	}
}

func TestRunCommandSilentUsesInjectedRunner(t *testing.T) {
	wantErr := errors.New("boom")
	runner := &fakeRunner{runErr: wantErr}
	restore := SetCommandRunner(runner)
	defer restore()

	err := RunCommandSilent("systemctl", "stop", "mihomo")
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunCommandSilent() error = %v, want %v", err, wantErr)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner call count = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "systemctl" {
		t.Fatalf("runner call name = %q, want %q", runner.calls[0].name, "systemctl")
	}
}

func TestSetCommandRunnerNilRestoresExecRunner(t *testing.T) {
	restore := SetCommandRunner(nil)
	defer restore()

	if _, ok := defaultRunner.(ExecRunner); !ok {
		t.Fatalf("defaultRunner type = %T, want ExecRunner", defaultRunner)
	}
}
