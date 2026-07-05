package input

import (
	"net/url"
	"strings"
)

// Provider identifies the external music source inferred from an input URL.
type Provider string

// ClassifiedURLs groups flattened input URLs by source provider.
type ClassifiedURLs struct {
	// Zvuk lists URLs classified as Zvuk provider inputs.
	Zvuk []string
	// Yandex lists URLs classified as Yandex Music provider inputs.
	Yandex []string
	// Unknown lists URLs that could not be classified to a known provider.
	Unknown []string
}

const (
	// ProviderZvuk identifies zvuk.com URLs.
	ProviderZvuk Provider = "zvuk"
	// ProviderYandex identifies music.yandex.* URLs.
	ProviderYandex Provider = "yandex"
	// ProviderUnknown is used for unsupported URLs.
	ProviderUnknown Provider = "unknown"
)

// Classify groups URLs by provider while preserving input order inside each group.
func Classify(values []string) *ClassifiedURLs {
	result := new(ClassifiedURLs)

	for _, value := range values {
		switch ClassifyURL(value) {
		case ProviderZvuk:
			result.Zvuk = append(result.Zvuk, value)
		case ProviderYandex:
			result.Yandex = append(result.Yandex, value)
		default:
			result.Unknown = append(result.Unknown, value)
		}
	}

	return result
}

// ClassifyURL infers provider from a URL host.
func ClassifyURL(raw string) Provider {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ProviderUnknown
	}

	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" {
		return providerFromHost(parsed.Hostname())
	}

	// Allow users to paste host/path without a scheme, e.g. music.yandex.ru/album/1.
	parsed, err = url.Parse("https://" + raw)
	if err == nil && parsed.Host != "" {
		return providerFromHost(parsed.Hostname())
	}

	return ProviderUnknown
}

// providerFromHost maps a hostname to a Provider value.
func providerFromHost(host string) Provider {
	host = strings.ToLower(strings.TrimSpace(host))

	switch {
	case host == "zvuk.com" || strings.HasSuffix(host, ".zvuk.com"):
		return ProviderZvuk
	case strings.HasPrefix(host, "music.yandex."):
		return ProviderYandex
	default:
		return ProviderUnknown
	}
}
