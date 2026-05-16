package inspect

import (
	"strings"
	"testing"
)

func TestInspectRequiresTarget(t *testing.T) {
	command := New()
	command.SetArgs([]string{})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "provide --id or --uuid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInspectRejectsMultipleTargets(t *testing.T) {
	command := New()
	command.SetArgs([]string{"--id", "0", "--uuid", "GPU-0"})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Fatalf("unexpected error: %v", err)
	}
}
