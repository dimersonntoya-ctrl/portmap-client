package wireguard

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"portmap.io/client/internal/config"
)

type Manager struct {
	config        *config.WireguardConfig
	device        *device.Device
	interfaceName string
}

func NewManager(config *config.WireguardConfig) *Manager {
	return &Manager{config: config}
}

func (m *Manager) Setup() error {
	dev, name, err := m.setupWireguardDevice()
	if err != nil {
		return err
	}
	m.device = dev
	m.interfaceName = name

	if err := m.configureIPAddress(); err != nil {
		return err
	}

	if err := m.addRoutes(); err != nil {
		return err
	}

	// Get interface IP for display
	// ip, _, _ := net.ParseCIDR(m.config.Interface.Address)
	// fmt.Printf("\n✓ WireGuard connection established\n")
	// fmt.Printf("  Interface: %s (%s)\n", m.interfaceName, ip)
	// fmt.Printf("  Press Ctrl+C to disconnect\n\n")

	return nil
}

func getTunnelName() string {
	switch runtime.GOOS {
	case "darwin":
		return "utun"
	case "windows":
		return "wg0"
	default:
		return "wg0"
	}
}

func convertKey(b64Key string) (string, error) {
	b64Key = strings.TrimSpace(b64Key)
	keyBytes, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 key: %v", err)
	}
	return hex.EncodeToString(keyBytes), nil
}

func (m *Manager) setupWireguardDevice() (*device.Device, string, error) {
	tunnelName := getTunnelName()

	tunDevice, err := tun.CreateTUN(tunnelName, device.DefaultMTU)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create TUN device on %s: %v", runtime.GOOS, err)
	}

	actualName, err := tunDevice.Name()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get interface name: %v", err)
	}

	bind := conn.NewDefaultBind()
	logger := device.NewLogger(device.LogLevelSilent, fmt.Sprintf("wireguard-%s", runtime.GOOS))

	dev := device.NewDevice(tunDevice, bind, logger)

	privateKey, err := convertKey(m.config.Interface.PrivateKey)
	if err != nil {
		return nil, "", fmt.Errorf("invalid private key: %v", err)
	}

	publicKey, err := convertKey(m.config.Peer.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("invalid public key: %v", err)
	}

	uapiConfig := fmt.Sprintf("private_key=%s\n"+
		"public_key=%s\n"+
		"endpoint=%s\n"+
		"allowed_ip=%s\n"+
		"persistent_keepalive_interval=%d\n",
		privateKey,
		publicKey,
		strings.TrimSpace(m.config.Peer.Endpoint),
		strings.TrimSpace(m.config.Peer.AllowedIPs[0]),
		m.config.Peer.PersistentKeepalive,
	)

	if err := dev.IpcSet(uapiConfig); err != nil {
		return nil, "", fmt.Errorf("failed to configure device: %v", err)
	}

	dev.Up()
	return dev, actualName, nil
}

func (m *Manager) configureIPAddress() error {
	ip, ipNet, err := net.ParseCIDR(m.config.Interface.Address)
	if err != nil {
		return fmt.Errorf("invalid IP address format: %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		peerIP := make(net.IP, len(ip))
		copy(peerIP, ip)
		mask := ipNet.Mask
		maskStr := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
		cmd := exec.Command("ifconfig", m.interfaceName, "inet", ip.String(), peerIP.String(), "netmask", maskStr)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set IP address: %v: %s", err, out)
		}
	case "linux":
		// First bring up the interface
		upCmd := exec.Command("ip", "link", "set", m.interfaceName, "up")
		if out, err := upCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to bring up interface: %v: %s", err, out)
		}

		// Then add the IP address
		addrCmd := exec.Command("ip", "addr", "add", "dev", m.interfaceName, m.config.Interface.Address)
		if out, err := addrCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set IP address: %v: %s", err, out)
		}
	case "windows":
		mask := ipNet.Mask
		maskStr := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
		cmd := exec.Command("netsh", "interface", "ip", "set", "address", m.interfaceName, "static", ip.String(), maskStr)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set IP address: %v: %s", err, out)
		}
	}
	return nil
}

func (m *Manager) addRoutes() error {
	for _, allowedIP := range m.config.Peer.AllowedIPs {
		if !isValidIP(allowedIP) {
			return fmt.Errorf("invalid IP address: %s", allowedIP)
		}
		if !isValidInterfaceName(m.interfaceName) {
			return fmt.Errorf("invalid interface name: %s", m.interfaceName)
		}
		switch runtime.GOOS {
		case "darwin":
			cmd := exec.Command("route", "add", "-net", allowedIP, "-interface", m.interfaceName)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to add route %s: %v: %s", allowedIP, err, out)
			}
		case "linux":
			cmd := exec.Command("ip", "route", "add", allowedIP, "dev", m.interfaceName)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to add route %s: %v: %s", allowedIP, err, out)
			}
		case "windows":
			// Get interface index first
			indexCmd := exec.Command("netsh", "interface", "ipv4", "show", "interfaces")
			output, err := indexCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to get interface index: %v: %s", err, output)
			}

			// Parse output to find interface index
			lines := strings.Split(string(output), "\n")
			var idx string
			for _, line := range lines {
				if strings.Contains(line, m.interfaceName) {
					fields := strings.Fields(line)
					if len(fields) > 0 {
						idx = fields[0]
						break
					}
				}
			}

			if idx == "" {
				return fmt.Errorf("could not find interface index for %s", m.interfaceName)
			}

			_, ipNet, err := net.ParseCIDR(allowedIP)
			if err != nil {
				return fmt.Errorf("invalid CIDR format: %v", err)
			}
			network := ipNet.IP.String()
			mask := net.IP(ipNet.Mask).String()
			gateway := strings.Split(m.config.Interface.Address, "/")[0]
			cmd := exec.Command("route", "add", network, "mask", mask, gateway, "metric", "1", "IF", idx)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to add route %s: %v: %s", allowedIP, err, out)
			}
		}
	}
	return nil
}

func (m *Manager) Cleanup() {
	if m.device != nil {
		m.device.Close()
	}

	switch runtime.GOOS {
	case "darwin":
		exec.Command("ifconfig", m.interfaceName, "down").Run()
	case "linux":
		exec.Command("ip", "link", "del", m.interfaceName).Run()
	case "windows":
		exec.Command("netsh", "interface", "delete", m.interfaceName).Run()
	}
}

func (m *Manager) GetTrafficStats() (rx uint64, tx uint64) {
	if m.device == nil {
		return 0, 0
	}

	// Get device stats using IpcGet
	stats, err := m.device.IpcGet()
	if err != nil {
		return 0, 0
	}

	// Parse stats from UAPI output
	for _, line := range strings.Split(stats, "\n") {
		if strings.HasPrefix(line, "rx_bytes=") {
			rx, _ = strconv.ParseUint(strings.TrimPrefix(line, "rx_bytes="), 10, 64)
		} else if strings.HasPrefix(line, "tx_bytes=") {
			tx, _ = strconv.ParseUint(strings.TrimPrefix(line, "tx_bytes="), 10, 64)
		}
	}

	return rx, tx
}

func (m *Manager) GetInterfaceName() string {
	return m.interfaceName
}

// isValidIP checks if the given string is a valid IP/CIDR
func isValidIP(ipStr string) bool {
	_, _, err := net.ParseCIDR(ipStr)
	if err == nil {
		return true
	}
	ip := net.ParseIP(ipStr)
	return ip != nil
}

// isValidInterfaceName checks if interface name matches allowed pattern
func isValidInterfaceName(name string) bool {
	// Match common interface naming patterns:
	// - utun[0-9]+
	// - wg[0-9]+
	// - tun[0-9]+
	matched, _ := regexp.MatchString(`^(utun|wg|tun)[0-9]+$`, name)
	return matched
}
