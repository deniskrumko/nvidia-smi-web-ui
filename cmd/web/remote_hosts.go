package web

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	webapp "github.com/deniskrumko/nvidia-smi-web-ui/app/web"
)

const remoteHostEnvPrefix = "REMOTE_HOST_"
const defaultRemoteHostPath = "/api/gpus"

type remoteHostEnv struct {
	displayName     string
	hostName        string
	path            string
	isDefault       bool
	seenDisplayName bool
	seenHostName    bool
	seenConfig      bool
}

func remoteHostsFromEnv() ([]webapp.RemoteHost, error) {
	return remoteHostsFromValues(os.Environ())
}

func remoteHostsFromValues(environ []string) ([]webapp.RemoteHost, error) {
	values := map[int]*remoteHostEnv{}
	for _, env := range environ {
		key, value, ok := strings.Cut(env, "=")
		if !ok || !strings.HasPrefix(key, remoteHostEnvPrefix) {
			continue
		}

		index, field, ok := splitRemoteHostKey(key)
		if !ok {
			continue
		}

		config := values[index]
		if config == nil {
			config = &remoteHostEnv{}
			values[index] = config
		}
		config.seenConfig = true

		switch field {
		case "DISPLAY_NAME":
			config.displayName = strings.TrimSpace(value)
			config.seenDisplayName = true
		case "HOST_NAME":
			config.hostName = strings.TrimSpace(value)
			config.seenHostName = true
		case "PATH":
			config.path = strings.TrimSpace(value)
		case "DEFAULT":
			config.isDefault = truthy(value)
		}
	}

	if len(values) == 0 {
		return nil, nil
	}

	indexes := make([]int, 0, len(values))
	for index := range values {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	hosts := make([]webapp.RemoteHost, 0, len(indexes))
	defaultIndex := -1
	for _, index := range indexes {
		if index != len(hosts) {
			return nil, fmt.Errorf("REMOTE_HOST indexes must start at 0 and be contiguous")
		}

		config := values[index]
		if !config.seenConfig {
			continue
		}
		if !config.seenDisplayName || config.displayName == "" {
			return nil, fmt.Errorf("REMOTE_HOST_%d_DISPLAY_NAME is required", index)
		}
		if !config.seenHostName || config.hostName == "" {
			return nil, fmt.Errorf("REMOTE_HOST_%d_HOST_NAME is required", index)
		}

		hostPath, err := normalizeRemoteHostPath(config.path)
		if err != nil {
			return nil, fmt.Errorf("REMOTE_HOST_%d_PATH: %w", index, err)
		}

		hostURL, err := normalizeRemoteHostURL(config.hostName, hostPath)
		if err != nil {
			return nil, fmt.Errorf("REMOTE_HOST_%d_HOST_NAME: %w", index, err)
		}
		if config.isDefault {
			if defaultIndex >= 0 {
				return nil, fmt.Errorf("only one REMOTE_HOST_*_DEFAULT can be true")
			}
			defaultIndex = len(hosts)
		}

		hosts = append(hosts, webapp.RemoteHost{
			Name:     config.displayName,
			HostName: config.hostName,
			URL:      hostURL,
			Default:  config.isDefault,
		})
	}

	if len(hosts) > 0 && defaultIndex < 0 {
		hosts[0].Default = true
	}

	return hosts, nil
}

func splitRemoteHostKey(key string) (int, string, bool) {
	rest := strings.TrimPrefix(key, remoteHostEnvPrefix)
	indexText, field, ok := strings.Cut(rest, "_")
	if !ok || indexText == "" || field == "" {
		return 0, "", false
	}

	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		return 0, "", false
	}

	return index, field, true
}

func normalizeRemoteHostURL(hostName string, endpointPath string) (string, error) {
	text := strings.TrimSpace(hostName)
	if text == "" {
		return "", fmt.Errorf("must not be empty")
	}
	if !strings.Contains(text, "://") {
		text = "https://" + text
	}

	parsed, err := url.Parse(text)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("must include a host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("must not include a path; use REMOTE_HOST_*_PATH")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("must not include query or fragment")
	}

	parsed.Path = endpointPath

	return parsed.String(), nil
}

func normalizeRemoteHostPath(path string) (string, error) {
	endpointPath := strings.TrimSpace(path)
	if endpointPath == "" {
		endpointPath = defaultRemoteHostPath
	}
	if !strings.HasPrefix(endpointPath, "/") {
		return "", fmt.Errorf("path must start with /")
	}
	if strings.ContainsAny(endpointPath, "?#") {
		return "", fmt.Errorf("path must not include query or fragment")
	}

	parsedPath, err := url.Parse(endpointPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if parsedPath.IsAbs() || parsedPath.Host != "" {
		return "", fmt.Errorf("path must be relative to the host")
	}

	return parsedPath.Path, nil
}

func truthy(value string) bool {
	text := strings.TrimSpace(value)
	return text == "1" || strings.EqualFold(text, "true") || strings.EqualFold(text, "yes")
}
