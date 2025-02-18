package wireguard

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
	"portmap.io/client/internal/config"
)

func ParseConfig(path string) (*config.WireguardConfig, string, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config: %v", err)
	}

	var config config.WireguardConfig

	// Get config_id from portmap section
	configID := cfg.Section("portmap").Key("config_id").String()
	if configID == "" {
		return nil, "", fmt.Errorf("config_id not found in [portmap] section")
	}

	// Parse Interface section
	interfaceSection := cfg.Section("Interface")
	config.Interface.PrivateKey = interfaceSection.Key("PrivateKey").String()
	config.Interface.Address = interfaceSection.Key("Address").String()
	config.Interface.DNS = interfaceSection.Key("DNS").String()

	// Parse Peer section
	peerSection := cfg.Section("Peer")
	config.Peer.PublicKey = peerSection.Key("PublicKey").String()
	config.Peer.AllowedIPs = strings.Split(peerSection.Key("AllowedIPs").String(), ",")
	config.Peer.Endpoint = peerSection.Key("Endpoint").String()
	config.Peer.PersistentKeepalive = peerSection.Key("PersistentKeepalive").MustInt(25)

	// Validate required fields
	if config.Interface.PrivateKey == "" {
		return nil, "", fmt.Errorf("missing private key")
	}
	if config.Interface.Address == "" {
		return nil, "", fmt.Errorf("missing interface address")
	}
	if config.Peer.PublicKey == "" {
		return nil, "", fmt.Errorf("missing peer public key")
	}
	if config.Peer.Endpoint == "" {
		return nil, "", fmt.Errorf("missing peer endpoint")
	}
	if len(config.Peer.AllowedIPs) == 0 {
		return nil, "", fmt.Errorf("missing allowed IPs")
	}

	return &config, configID, nil
}
