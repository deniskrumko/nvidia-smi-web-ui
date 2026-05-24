package web

import (
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
		{name: "empty defaults enabled", value: "", want: false},
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
		"REMOTE_HOST_0_DISPLAY_NAME=lab",
		"REMOTE_HOST_0_HOST_NAME=nvidia-web-ui-rnd-kube.kolesa-team.org",
		"REMOTE_HOST_1_DISPLAY_NAME=stage",
		"REMOTE_HOST_1_HOST_NAME=http://127.0.0.1:9090",
		"REMOTE_HOST_1_PATH=/api/custom-gpus",
		"REMOTE_HOST_1_DEFAULT=true",
	})
	if err != nil {
		t.Fatalf("parse remote hosts: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Name != "lab" || hosts[0].HostName != "nvidia-web-ui-rnd-kube.kolesa-team.org" || hosts[0].URL != "https://nvidia-web-ui-rnd-kube.kolesa-team.org/api/gpus" || hosts[0].Default {
		t.Fatalf("unexpected first host: %#v", hosts[0])
	}
	if hosts[1].Name != "stage" || hosts[1].HostName != "http://127.0.0.1:9090" || hosts[1].URL != "http://127.0.0.1:9090/api/custom-gpus" || !hosts[1].Default {
		t.Fatalf("unexpected second host: %#v", hosts[1])
	}
}

func TestRemoteHostsFromEnvDefaultsToFirstHost(t *testing.T) {
	hosts, err := remoteHostsFromValues([]string{
		"REMOTE_HOST_0_DISPLAY_NAME=lab",
		"REMOTE_HOST_0_HOST_NAME=https://example.test",
	})
	if err != nil {
		t.Fatalf("parse remote hosts: %v", err)
	}
	if len(hosts) != 1 || !hosts[0].Default {
		t.Fatalf("expected first host to be default, got %#v", hosts)
	}
}

func TestRemoteHostsFromEnvRequiresDisplayNameAndHostName(t *testing.T) {
	_, err := remoteHostsFromValues([]string{"REMOTE_HOST_0_DISPLAY_NAME=lab"})
	if err == nil {
		t.Fatal("expected missing host name error")
	}
}

func TestRemoteHostsFromEnvRequiresContiguousIndexes(t *testing.T) {
	_, err := remoteHostsFromValues([]string{
		"REMOTE_HOST_1_DISPLAY_NAME=stage",
		"REMOTE_HOST_1_HOST_NAME=https://example.test",
	})
	if err == nil {
		t.Fatal("expected contiguous indexes error")
	}
}

func TestRemoteHostsFromEnvRejectsPathInHostName(t *testing.T) {
	_, err := remoteHostsFromValues([]string{
		"REMOTE_HOST_0_DISPLAY_NAME=lab",
		"REMOTE_HOST_0_HOST_NAME=https://example.test/api/gpus",
	})
	if err == nil {
		t.Fatal("expected path in host name error")
	}
}

func TestRemoteHostsFromEnvRequiresPathPrefix(t *testing.T) {
	_, err := remoteHostsFromValues([]string{
		"REMOTE_HOST_0_DISPLAY_NAME=lab",
		"REMOTE_HOST_0_HOST_NAME=https://example.test",
		"REMOTE_HOST_0_PATH=api/gpus",
	})
	if err == nil {
		t.Fatal("expected invalid path error")
	}
}

func TestNormalizeRemoteHostURL(t *testing.T) {
	tests := []struct {
		name         string
		hostName     string
		endpointPath string
		want         string
		wantErr      bool
	}{
		{
			name:         "adds default https scheme",
			hostName:     "example.test",
			endpointPath: "/api/gpus",
			want:         "https://example.test/api/gpus",
		},
		{
			name:         "keeps http scheme and port",
			hostName:     "http://127.0.0.1:9090",
			endpointPath: "/api/gpus",
			want:         "http://127.0.0.1:9090/api/gpus",
		},
		{
			name:         "trims host name",
			hostName:     " https://example.test ",
			endpointPath: "/api/gpus",
			want:         "https://example.test/api/gpus",
		},
		{
			name:         "allows root path in host name",
			hostName:     "https://example.test/",
			endpointPath: "/api/gpus",
			want:         "https://example.test/api/gpus",
		},
		{
			name:         "uses custom endpoint path",
			hostName:     "https://example.test",
			endpointPath: "/api/custom-gpus",
			want:         "https://example.test/api/custom-gpus",
		},
		{
			name:         "rejects empty host name",
			hostName:     "",
			endpointPath: "/api/gpus",
			wantErr:      true,
		},
		{
			name:         "rejects unsupported scheme",
			hostName:     "ftp://example.test",
			endpointPath: "/api/gpus",
			wantErr:      true,
		},
		{
			name:         "rejects missing host",
			hostName:     "https:///example.test",
			endpointPath: "/api/gpus",
			wantErr:      true,
		},
		{
			name:         "rejects path in host name",
			hostName:     "https://example.test/api/gpus",
			endpointPath: "/api/gpus",
			wantErr:      true,
		},
		{
			name:         "rejects query in host name",
			hostName:     "https://example.test?token=abc",
			endpointPath: "/api/gpus",
			wantErr:      true,
		},
		{
			name:         "rejects fragment in host name",
			hostName:     "https://example.test#gpu",
			endpointPath: "/api/gpus",
			wantErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeRemoteHostURL(test.hostName, test.endpointPath)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}
