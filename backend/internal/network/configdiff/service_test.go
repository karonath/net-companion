package configdiff

import "testing"

type mockRunner struct {
	outputs map[string]string
}

func (m mockRunner) Run(cmd string) (string, error) {
	return m.outputs[cmd], nil
}

func TestFetchRunsBothCommands(t *testing.T) {
	r := mockRunner{outputs: map[string]string{
		cmdRunning: "hostname R1\nno shutdown",
		cmdStartup: "hostname R1",
	}}
	running, startup, err := Fetch(r)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if running != "hostname R1\nno shutdown" || startup != "hostname R1" {
		t.Fatalf("running=%q startup=%q", running, startup)
	}
	// et le diff enchaîne proprement
	lines := Diff(startup, running)
	if len(lines) != 2 || lines[1].Op != "add" || lines[1].Text != "no shutdown" {
		t.Fatalf("diff = %+v", lines)
	}
}
