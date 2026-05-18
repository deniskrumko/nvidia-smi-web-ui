package web

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

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
			t.Setenv(debugModeEnv, test.value)

			if got := debugEnabled(); got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestAccessLogEnabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty defaults enabled", value: "", want: true},
		{name: "true", value: "true", want: true},
		{name: "yes", value: "yes", want: true},
		{name: "one", value: "1", want: true},
		{name: "false", value: "false", want: false},
		{name: "zero", value: "0", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(accessLogEnv, test.value)

			if got := accessLogEnabled(); got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestAccessLogLevelFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: ""},
		{name: "debug", value: "debug"},
		{name: "info", value: "info"},
		{name: "warn", value: "warn"},
		{name: "warning", value: "warning"},
		{name: "error", value: "error"},
		{name: "invalid", value: "trace", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := accessLogLevelFromEnv(test.value)
			if test.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestNewLogHandlerUsesJSON(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(newLogHandler(&output))

	logger.Info("Serving web UI at", "url", "http://localhost:8080")

	log := output.String()
	for _, expected := range []string{
		`"level":"INFO"`,
		`"msg":"Serving web UI at"`,
		`"url":"http://localhost:8080"`,
	} {
		if !strings.Contains(log, expected) {
			t.Fatalf("expected JSON log to contain %s, got %q", expected, log)
		}
	}
}

func TestServingMode(t *testing.T) {
	tests := []struct {
		name   string
		suffix string
		want   string
	}{
		{name: "empty", suffix: "", want: "local GPU data"},
		{name: "debug", suffix: " with debug GPU data and NVML disabled", want: "debug GPU data and NVML disabled"},
		{name: "remote", suffix: " with remote GPU hosts", want: "remote GPU hosts"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := servingMode(test.suffix); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestRemoteHostsFromEnv(t *testing.T) {
	hosts, err := remoteHostsFromValues([]string{
		"REMOTE_HOST_0_NAME=lab",
		"REMOTE_HOST_0_URL=nvidia-web-ui-rnd-kube.kolesa-team.org/api/gpus",
		"REMOTE_HOST_1_NAME=stage",
		"REMOTE_HOST_1_URL=http://127.0.0.1:9090/api/gpus",
		"REMOTE_HOST_1_DEFAULT=true",
	})
	if err != nil {
		t.Fatalf("parse remote hosts: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Name != "lab" || hosts[0].URL != "https://nvidia-web-ui-rnd-kube.kolesa-team.org/api/gpus" || hosts[0].Default {
		t.Fatalf("unexpected first host: %#v", hosts[0])
	}
	if hosts[1].Name != "stage" || hosts[1].URL != "http://127.0.0.1:9090/api/gpus" || !hosts[1].Default {
		t.Fatalf("unexpected second host: %#v", hosts[1])
	}
}

func TestRemoteHostsFromEnvDefaultsToFirstHost(t *testing.T) {
	hosts, err := remoteHostsFromValues([]string{
		"REMOTE_HOST_0_NAME=lab",
		"REMOTE_HOST_0_URL=https://example.test/api/gpus",
	})
	if err != nil {
		t.Fatalf("parse remote hosts: %v", err)
	}
	if len(hosts) != 1 || !hosts[0].Default {
		t.Fatalf("expected first host to be default, got %#v", hosts)
	}
}

func TestRemoteHostsFromEnvRequiresNameAndURL(t *testing.T) {
	_, err := remoteHostsFromValues([]string{"REMOTE_HOST_0_NAME=lab"})
	if err == nil {
		t.Fatal("expected missing URL error")
	}
}

func TestRemoteHostsFromEnvRequiresContiguousIndexes(t *testing.T) {
	_, err := remoteHostsFromValues([]string{
		"REMOTE_HOST_1_NAME=stage",
		"REMOTE_HOST_1_URL=https://example.test/api/gpus",
	})
	if err == nil {
		t.Fatal("expected contiguous indexes error")
	}
}
