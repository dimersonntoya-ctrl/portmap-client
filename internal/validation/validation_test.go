package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidations(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"Hostname", testHostnameValidation},
		{"Port", testPortValidation},
		{"PortNumber", testPortNumberValidation},
		{"Protocol", testProtocolValidation},
		{"HostHeader", testHostHeaderValidation},
		{"CIDR", testCIDRValidation},
		{"WSTimeout", testWSTimeoutValidation},
		{"ConfigType", testConfigTypeValidation},
		{"Region", testRegionValidation},
		{"OpenVPNProto", testOpenVPNProtoValidation},
		{"Name", testNameValidation},
		{"Comment", testCommentValidation},
		{"ID", testIDValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testHostnameValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"test.portmap.io", true, ""},
		{"test.portmap.host", true, ""},
		{"invalid.domain", false, "Hostname must end with .portmap.io or .portmap.host"},
		{"", false, "Invalid hostname format"},
		{"test@.portmap.io", false, "Invalid hostname format"},
		{strings.Repeat("a", 51) + ".portmap.io", false, "Hostname must not exceed 50 characters"},
	}

	for _, tt := range tests {
		valid, msg := IsValidHostname(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testPortValidation(t *testing.T) {
	tests := []struct {
		port       string
		protocol   string
		configType string
		isValid    bool
		errorMsg   string
	}{
		{"8080", "tcp", "OpenVPN", true, ""},
		{"80", "http", "OpenVPN", true, ""},
		{"443", "https", "OpenVPN", true, ""},
		{"80", "tcp", "OpenVPN", false, "Port 80 is only allowed for http protocol"},
		{"443", "tcp", "OpenVPN", false, "Port 443 is only allowed for https protocol"},
		{"1194", "tcp", "OpenVPN", false, "Port 1194 is not allowed (OpenVPN default port)"},
		{"0", "tcp", "OpenVPN", false, "Port must be in range [1024-65535], except for special cases (80/443)"},
		{"65536", "tcp", "OpenVPN", false, "Port must be in range [1024-65535], except for special cases (80/443)"},
	}

	for _, tt := range tests {
		valid, msg := IsValidPort(tt.port, tt.protocol, tt.configType)
		assert.Equal(t, tt.isValid, valid, "port: %s, protocol: %s, configType: %s", tt.port, tt.protocol, tt.configType)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testPortNumberValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"1", true, ""},
		{"65535", true, ""},
		{"0", false, "Port must be in range [1-65535]"},
		{"65536", false, "Port must be in range [1-65535]"},
		{"abc", false, "Port must be a number"},
	}

	for _, tt := range tests {
		valid, msg := IsValidPortNumber(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testProtocolValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"tcp", true, ""},
		{"udp", true, ""},
		{"http", true, ""},
		{"https", true, ""},
		{"invalid", false, "Protocol must be one of: tcp, udp, http, https"},
	}

	for _, tt := range tests {
		valid, msg := IsValidProtocol(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testHostHeaderValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"example.com", true, ""},
		{"", true, ""},
		{"invalid@host", false, "Invalid host header format"},
		{strings.Repeat("a", 51), false, "Host header must not exceed 50 characters"},
	}

	for _, tt := range tests {
		valid, msg := IsValidHostHeader(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testCIDRValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"192.168.1.0/24", true, ""},
		{"", true, ""},
		{"invalid", false, "Invalid CIDR format (e.g., 192.168.1.0/24)"},
		{"256.256.256.256/24", false, "Invalid CIDR format (e.g., 192.168.1.0/24)"},
	}

	for _, tt := range tests {
		valid, msg := IsValidCIDR(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testWSTimeoutValidation(t *testing.T) {
	tests := []struct {
		input    int
		isValid  bool
		errorMsg string
	}{
		{30, true, ""},
		{0, true, ""},
		{3600, true, ""},
		{-1, false, "WebSocket timeout must be non-negative"},
		{3601, false, "WebSocket timeout must not exceed 3600 seconds (1 hour)"},
	}

	for _, tt := range tests {
		valid, msg := IsValidWSTimeout(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %d", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testConfigTypeValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"OpenVPN", true, ""},
		{"SSH", true, ""},
		{"WireGuard", true, ""},
		{"invalid", false, "Type must be one of: OpenVPN, SSH, WireGuard"},
	}

	for _, tt := range tests {
		valid, msg := IsValidConfigType(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testRegionValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"default", true, ""},
		{"nyc1", true, ""},
		{"fra1", true, ""},
		{"blr1", true, ""},
		{"sin1", true, ""},
		{"invalid", false, "Region must be one of: default, nyc1, fra1, blr1, sin1"},
	}

	for _, tt := range tests {
		valid, msg := IsValidRegion(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testOpenVPNProtoValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"tcp", true, ""},
		{"udp", true, ""},
		{"invalid", false, "OpenVPN protocol must be one of: tcp, udp"},
	}

	for _, tt := range tests {
		valid, msg := IsValidOpenVPNProto(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testNameValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"test-config", true, ""},
		{"test123", true, ""},
		{"", false, "Name must contain only letters, numbers, and hyphens, and must start and end with alphanumeric character"},
		{"test@config", false, "Name must contain only letters, numbers, and hyphens, and must start and end with alphanumeric character"},
		{strings.Repeat("a", 51), false, "Name must not exceed 50 characters"},
	}

	for _, tt := range tests {
		valid, msg := IsValidName(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testCommentValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"Test comment", true, ""},
		{"", true, ""},
		{strings.Repeat("a", 256), false, "Comment must not exceed 255 characters"},
	}

	for _, tt := range tests {
		valid, msg := IsValidComment(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}

func testIDValidation(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		errorMsg string
	}{
		{"123456", true, ""},
		{"", false, "ID cannot be empty"},
		{"abc123", false, "ID must contain only numbers"},
		{"123456789", false, "ID must be less than 8 digits"},
	}

	for _, tt := range tests {
		valid, msg := IsValidID(tt.input)
		assert.Equal(t, tt.isValid, valid, "input: %s", tt.input)
		if !tt.isValid {
			assert.Equal(t, tt.errorMsg, msg)
		}
	}
}
