package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

const (
	MaxInputLength   = 50
	MaxCommentLength = 255
)

func validateLength(input string, maxLen int, fieldName string) (bool, string) {
	if len(input) > maxLen {
		return false, fmt.Sprintf("%s must not exceed %d characters", fieldName, maxLen)
	}
	return true, ""
}

// IsValidHostname validates a hostname against portmap.io requirements
func IsValidHostname(hostname string) (bool, string) {
	if valid, msg := validateLength(hostname, MaxInputLength, "Hostname"); !valid {
		return false, msg
	}
	validDomains := []string{".portmap.io", ".portmap.host"}
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

	if !domainRegex.MatchString(hostname) {
		return false, "Invalid hostname format"
	}

	for _, domain := range validDomains {
		if strings.HasSuffix(hostname, domain) {
			return true, ""
		}
	}

	return false, "Hostname must end with .portmap.io or .portmap.host"
}

// IsValidConfigID validates a config ID against available configurations
func IsValidConfigID(configID string, configs map[string]interface{}) bool {
	if data, ok := configs["data"].([]interface{}); ok {
		for _, conf := range data {
			if config, ok := conf.(map[string]interface{}); ok {
				if fmt.Sprintf("%v", config["id"]) == configID {
					return true
				}
			}
		}
	}
	return false
}

// IsValidPort validates a port number against protocol and config type restrictions
func IsValidPort(port string, protocol string, configType string) (bool, string) {
	portNum := 0
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		return false, "Port must be a number"
	}

	reservedPorts := map[int]string{
		1194: "OpenVPN default port",
		3306: "MySQL default port",
	}
	if reason, reserved := reservedPorts[portNum]; reserved {
		return false, fmt.Sprintf("Port %d is not allowed (%s)", portNum, reason)
	}

	if portNum == 80 {
		if protocol != "http" {
			return false, "Port 80 is only allowed for http protocol"
		}
		if configType != "OpenVPN" && configType != "WireGuard" {
			return false, "Port 80 is only allowed for OpenVPN and WireGuard configurations"
		}
		return true, ""
	}

	if portNum == 443 {
		if protocol != "https" {
			return false, "Port 443 is only allowed for https protocol"
		}
		if configType != "OpenVPN" && configType != "WireGuard" {
			return false, "Port 443 is only allowed for OpenVPN and WireGuard configurations"
		}
		return true, ""
	}

	if portNum < 1024 || portNum > 65535 {
		return false, "Port must be in range [1024-65535], except for special cases (80/443)"
	}

	return true, ""
}

// IsValidPortNumber validates a general port number
func IsValidPortNumber(port string) (bool, string) {
	portNum := 0
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		return false, "Port must be a number"
	}

	if portNum < 1 || portNum > 65535 {
		return false, "Port must be in range [1-65535]"
	}

	return true, ""
}

// IsValidProtocol validates the protocol against allowed values
func IsValidProtocol(protocol string) (bool, string) {
	validProtocols := map[string]bool{
		"tcp":   true,
		"udp":   true,
		"http":  true,
		"https": true,
	}

	if _, ok := validProtocols[protocol]; !ok {
		return false, "Protocol must be one of: tcp, udp, http, https"
	}
	return true, ""
}

// IsValidHostHeader validates an optional host header
func IsValidHostHeader(header string) (bool, string) {
	if header == "" {
		return true, "" // Optional field
	}

	if valid, msg := validateLength(header, MaxInputLength, "Host header"); !valid {
		return false, msg
	}

	hostRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostRegex.MatchString(header) {
		return false, "Invalid host header format"
	}
	return true, ""
}

// IsValidCIDR validates an optional CIDR notation IP range
func IsValidCIDR(cidr string) (bool, string) {
	if cidr == "" {
		return true, "" // Optional field
	}

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, "Invalid CIDR format (e.g., 192.168.1.0/24)"
	}
	return true, ""
}

// IsValidWSTimeout validates WebSocket timeout value
func IsValidWSTimeout(timeout int) (bool, string) {
	if timeout < 0 {
		return false, "WebSocket timeout must be non-negative"
	}
	if timeout > 3600 {
		return false, "WebSocket timeout must not exceed 3600 seconds (1 hour)"
	}
	return true, ""
}

func IsValidConfigType(configType string) (bool, string) {
	if valid, msg := validateLength(configType, MaxInputLength, "Configuration type"); !valid {
		return false, msg
	}
	validTypes := map[string]bool{
		"OpenVPN":   true,
		"SSH":       true,
		"WireGuard": true,
	}

	if _, ok := validTypes[configType]; !ok {
		return false, "Type must be one of: OpenVPN, SSH, WireGuard"
	}
	return true, ""
}

func IsValidRegion(region string) (bool, string) {
	if valid, msg := validateLength(region, MaxInputLength, "Region"); !valid {
		return false, msg
	}
	validRegions := map[string]bool{
		"default": true,
		"nyc1":    true,
		"fra1":    true,
		"blr1":    true,
		"sin1":    true,
	}

	if _, ok := validRegions[region]; !ok {
		return false, "Region must be one of: default, nyc1, fra1, blr1, sin1"
	}
	return true, ""
}

func IsValidOpenVPNProto(proto string) (bool, string) {
	validProtos := map[string]bool{
		"tcp": true,
		"udp": true,
	}

	if _, ok := validProtos[proto]; !ok {
		return false, "OpenVPN protocol must be one of: tcp, udp"
	}
	return true, ""
}

func IsValidName(name string) (bool, string) {
	if valid, msg := validateLength(name, MaxInputLength, "Name"); !valid {
		return false, msg
	}
	// Basic name validation - alphanumeric and hyphens
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,48}[a-zA-Z0-9])?$`)
	if !nameRegex.MatchString(name) {
		return false, "Name must contain only letters, numbers, and hyphens, and must start and end with alphanumeric character"
	}
	return true, ""
}

func IsValidComment(comment string) (bool, string) {
	if comment == "" {
		return true, "" // Optional field
	}

	if valid, msg := validateLength(comment, MaxCommentLength, "Comment"); !valid {
		return false, msg
	}
	return true, ""
}

// Add after other validation functions
func IsValidID(id string) (bool, string) {
	// Check if string contains only digits
	for _, r := range id {
		if r < '0' || r > '9' {
			return false, "ID must contain only numbers"
		}
	}

	// Check length
	if len(id) > 8 {
		return false, "ID must be less than 8 digits"
	}

	if len(id) == 0 {
		return false, "ID cannot be empty"
	}

	return true, ""
}
