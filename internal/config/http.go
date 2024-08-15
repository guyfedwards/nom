package config

import (
	"crypto/tls"
	"fmt"
)

// CloudFlare blocks requests unless a minimum TLSVersion is specified.
var tlsVersions map[string]uint16 = map[string]uint16{
	"TLS 1.0": tls.VersionTLS10,
	"TLS 1.1": tls.VersionTLS11,
	"TLS 1.2": tls.VersionTLS12,
	"TLS 1.3": tls.VersionTLS13,
}

type HTTPOptions struct {
	//MinTLSVersion must be set to one of the strings returned by
	//tls.VersionName. "TLS 1.2" by default.
	MinTLSVersion string `yaml:"mintls,omitempty"`
}

// TLSVersion maps one of a few supported TLS version strings to the corresponding
// standard TLS library constant so that an HTTP client can be configured.
func TLSVersion(configStr string) (uint16, error) {
	if version, ok := tlsVersions[configStr]; ok {
		return version, nil
	}
	return 0, fmt.Errorf("unsupported tls version: %s", configStr)
}
