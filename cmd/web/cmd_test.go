package web

import "testing"

func TestDebugEnabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty", value: "", want: false},
		{name: "one", value: "1", want: true},
		{name: "true", value: "true", want: true},
		{name: "yes", value: "yes", want: true},
		{name: "false", value: "false", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(debugEnv, test.value)

			if got := debugEnabled(); got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}
