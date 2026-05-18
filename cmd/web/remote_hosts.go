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

type remoteHostEnv struct {
	name       string
	url        string
	isDefault  bool
	seenName   bool
	seenURL    bool
	seenConfig bool
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
		case "NAME":
			config.name = strings.TrimSpace(value)
			config.seenName = true
		case "URL":
			config.url = strings.TrimSpace(value)
			config.seenURL = true
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
		if !config.seenName || config.name == "" {
			return nil, fmt.Errorf("REMOTE_HOST_%d_NAME is required", index)
		}
		if !config.seenURL || config.url == "" {
			return nil, fmt.Errorf("REMOTE_HOST_%d_URL is required", index)
		}

		hostURL, err := normalizeRemoteHostURL(config.url)
		if err != nil {
			return nil, fmt.Errorf("REMOTE_HOST_%d_URL: %w", index, err)
		}
		if config.isDefault {
			if defaultIndex >= 0 {
				return nil, fmt.Errorf("only one REMOTE_HOST_*_DEFAULT can be true")
			}
			defaultIndex = len(hosts)
		}

		hosts = append(hosts, webapp.RemoteHost{
			Name:    config.name,
			URL:     hostURL,
			Default: config.isDefault,
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

func normalizeRemoteHostURL(value string) (string, error) {
	text := strings.TrimSpace(value)
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

	return parsed.String(), nil
}

func truthy(value string) bool {
	text := strings.TrimSpace(value)
	return text == "1" || strings.EqualFold(text, "true") || strings.EqualFold(text, "yes")
}
