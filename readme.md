# Portmap.io Client

Welcome to the first ever [Portmap.io](https://portmap.io) Client! This command-line tool allows you to easily manage your portmap.io mapping rules using the [portmap.io REST API](https://portmap.io/restapi). Whether you're setting up configurations, managing mapping rules, or connecting to a VPN, this client has you covered.

## Installation

Subscribe to the premium plan and get your API token from the [portmap.io profile page](https://portmap.io/profile).

Download the latest binary for your platform from the releases page:
### MacOS and Linux

```bash
# macOS
curl -Lo portmap https://github.com/portmap-io/portmap-client/releases/latest/download/portmap-darwin-arm64
chmod +x portmap
sudo mv portmap /usr/local/bin/

# Linux
curl -Lo portmap https://github.com/portmap-io/portmap-client/releases/latest/download/portmap-linux-amd64
chmod +x portmap
sudo mv portmap /usr/local/bin/
```
### Windows
Download portmap.exe from releases page
https://github.com/portmap-io/portmap-client/releases/latest/download/portmap-windows-amd64.exe
Download wintun.dll from [wintun.net](https://www.wintun.net/)

Place `wintun.dll` in the same directory as `portmap.exe`

Without `wintun.dll`, the WireGuard connection will fail to establish on Windows systems.

## Global Options

The following options are available for all commands:

- `--env-file`: Path to custom .env file (default: .env in current directory)
- `--output`: Output format (json/text)

Example:
```bash
# Use custom .env file
portmap --env-file=/path/to/custom.env mapping list

# Use default .env in current directory
portmap mapping list
```

## Commands

### Initialize Client

Configure the client with your API token:

```bash
portmap init
```

### Configuration Management

List configurations:
```bash
# Show all columns (default)
portmap config list

# Show specific columns
portmap config list --columns=id,name,type,region

# List configs with specific columns and filtering
portmap config list --type=WireGuard --columns=id,name,region,created_at

```



Show configuration details:
```bash
portmap config show [config-id]
```

Download configuration file or SSH key file:
```bash
portmap config show [config-id] --save-config
```

### Mapping rules management

List mapping rules:
```bash
# Show all columns (default)
portmap mapping list

# Show specific columns
portmap mapping list --columns=id,hostname,protocol,port_from,port_to

# List mappings with custom columns and region filter
portmap mapping list --region=fra1 --columns=hostname,protocol,port_from,port_to
```

Create new mapping:
```bash
portmap mapping create [flags]
```

Without flags it will prompt for all required fields.

Flags:
- `--config-id`: config ID
- `--hostname`: Hostname
- `--protocol`: Protocol (http, https, tcp, udp)
- `--port-from`: External port
- `--port-to`: Local port
- `--proxy-to-http`: Proxy HTTPS to HTTP (HTTPS only, default: true)
- `--region`: region (default from .env)

Example:
```bash
$ portmap mapping create --hostname test-rule12.portmap.io --config-id 123 --protocol https --port-from 443 --port-to 8080

```

Show mapping details:
```bash
portmap mapping show [mapping-id]
```

Delete mapping:
```bash
portmap mapping delete [mapping-id]
```

### Connect to WireGuard VPN

```bash
portmap connect [config-file]
```

Options:
- `--service`: Run in service mode with minimal output

Example:
```bash
$ sudo portmap connect wireguard.conf
✓ Connected to fra1.portmap.io via utun4
  
Press Ctrl+C to disconnect

✓ Available mapping rules:
  • https://app1.portmap.io:443 => http://10.0.0.2:80
  • http://app2.portmap.io:80 => http://10.0.0.2:80

↓ 0 B received
↑ 0 B sent
```

Service mode example:
```bash
$ portmap connect --service wireguard.conf
Connected to fra1.portmap.io via utun4
```

## Output Formats

The client supports two output formats (defaulted to one from .env):
- `json`: Machine-readable JSON output
- `text`: Human-friendly formatted text

Override format for single command:
```bash
portmap mapping list --output json
```

## Environment Variables

The client uses the following environment variables:

- `PORTMAP_TOKEN`: API token
- `PORTMAP_FORMAT`: Output format (json/text)
- `PORTMAP_REGION`: Default region

Example .env file:
```ini
PORTMAP_TOKEN=your-api-token-here
PORTMAP_FORMAT=text
PORTMAP_REGION=fra1
```

