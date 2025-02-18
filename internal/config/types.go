package config

type WireguardConfig struct {
    Interface struct {
        PrivateKey string
        Address    string
        DNS        string
    }
    Peer struct {
        PublicKey           string
        AllowedIPs         []string
        Endpoint           string
        PersistentKeepalive int
    }
}
