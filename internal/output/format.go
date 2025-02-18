package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

var writer io.Writer = os.Stdout

func SetWriter(w io.Writer) {
	writer = w
}

func GetWriter() io.Writer {
	return writer
}

type Format string

const (
	JSON Format = "json"
	Text Format = "text"
)

func ParseFormat(format string) (Format, error) {
	switch strings.ToLower(format) {
	case "json":
		return JSON, nil
	case "text":
		return Text, nil
	default:
		return "", fmt.Errorf("invalid output format: %s (supported: json, text)", format)
	}
}

// Add new type for column selection
type Options struct {
	Format  Format
	Columns []string
}

// Update Print function signature
func Print(data interface{}, opts Options) error {
	switch opts.Format {
	case JSON:
		return printJSON(data)
	case Text:
		return printText(data, opts.Columns)
	default:
		return fmt.Errorf("unsupported output format: %s", opts.Format)
	}
}

func printJSON(data interface{}) error {
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer, string(prettyJSON))
	return err
}

func printText(data interface{}, columns []string) error {
	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Extract data from wrapper if present
	if wrapper, ok := data.(map[string]interface{}); ok {
		if dataField, exists := wrapper["data"]; exists {
			data = dataField
		}
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return printSingleTable(w, v, columns)
	case []interface{}:
		if len(v) == 0 {
			fmt.Fprintln(w, "No data available")
			return nil
		}
		return printArrayTable(w, v, columns)
	default:
		fmt.Fprintf(w, "%v\n", v)
	}
	return nil
}

func printSingleTable(w *tabwriter.Writer, data map[string]interface{}, columns []string) error {
	// Print header
	fmt.Fprintln(w, "KEY\tVALUE")
	fmt.Fprintln(w, "---\t-----")

	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Print rows
	for _, k := range keys {
		if k == "config" {
			if config, ok := data[k].(map[string]interface{}); ok {
				name, _, _, id := extractConfigInfo(config)
				fmt.Fprintf(w, "%s\t%s\n", k, name)
				fmt.Fprintf(w, "config_id\t%s\n", id)
				continue
			}
		}
		fmt.Fprintf(w, "%s\t%s\n", k, formatValue(data[k]))
	}
	return nil
}

// Update printArrayTable to use columns
func printArrayTable(w *tabwriter.Writer, data []interface{}, columns []string) error {
	if len(data) == 0 {
		return nil
	}

	firstItem, ok := data[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("array items must be objects")
	}

	// Check if this is a config list or mapping list
	if _, hasConfig := firstItem["config"]; hasConfig {
		return printMappingTable(w, data, columns)
	}
	return printConfigTable(w, data, columns)
}

// Update printMappingTable to use columns
func printMappingTable(w *tabwriter.Writer, data []interface{}, requestedColumns []string) error {
	// Default columns if none specified
	defaultColumns := []string{
		"id",
		"region",
		"config_id",
		"config_name",
		"config_type",
		"hostname",
		"protocol",
		"port_from",
		"port_to",
		"created_at",
		"active",
	}

	// Use requested columns if provided, otherwise use defaults
	columnOrder := defaultColumns
	if len(requestedColumns) > 0 {
		columnOrder = requestedColumns
	}

	// Print headers
	fmt.Fprintln(w, strings.Join(columnOrder, "\t"))
	fmt.Fprintln(w, strings.Repeat("---\t", len(columnOrder)))

	// Print rows
	for _, item := range data {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		values := make([]string, len(columnOrder))
		for i, header := range columnOrder {
			switch header {
			case "config_name", "config_type", "region", "config_id": // Add config_id here
				if config, ok := row["config"].(map[string]interface{}); ok {
					name, typ, reg, id := extractConfigInfo(config)
					switch header {
					case "config_name":
						values[i] = name
					case "config_type":
						values[i] = typ
					case "region":
						values[i] = reg
					case "config_id":
						values[i] = id
					}
				} else {
					values[i] = "-"
				}
			default:
				values[i] = formatValue(row[header])
			}
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	return nil
}

func printConfigTable(w *tabwriter.Writer, data []interface{}, requestedColumns []string) error {
	// Default columns if none specified
	defaultColumns := []string{
		"id",
		"region",
		"name",
		"type",
		"proto",
		"created_at",
		"comment",
	}

	// Use requested columns if provided, otherwise use defaults
	columnOrder := defaultColumns
	if len(requestedColumns) > 0 {
		columnOrder = requestedColumns
	}

	// Print headers
	fmt.Fprintln(w, strings.Join(columnOrder, "\t"))
	fmt.Fprintln(w, strings.Repeat("---\t", len(columnOrder)))

	// Print rows
	for _, item := range data {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		values := make([]string, len(columnOrder))
		for i, header := range columnOrder {
			switch header {
			case "name", "type", "region", "proto", "comment":
				if val, exists := row[header]; exists {
					values[i] = formatValue(val)
				} else {
					values[i] = "-"
				}
			case "created_at":
				if val, exists := row[header]; exists {
					values[i] = formatValue(val)
				} else {
					values[i] = "-"
				}
			case "id":
				if val, exists := row[header]; exists {
					values[i] = formatValue(val)
				} else {
					values[i] = "-"
				}
			default:
				values[i] = formatValue(row[header])
			}
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	return nil
}

func extractConfigInfo(config map[string]interface{}) (name, configType, region, id string) {
	name = "-"
	configType = "-"
	region = "-"
	id = "-"

	if n, ok := config["name"].(string); ok {
		name = n
	}
	if t, ok := config["type"].(string); ok {
		configType = t
	}
	if r, ok := config["region"].(string); ok {
		region = r
	}
	if i, ok := config["id"].(float64); ok {
		id = fmt.Sprintf("%.0f", i) // Format as integer without decimal places
	}
	return
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		if t, err := parseDate(val); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
		return val
	case float64:
		if float64(int64(val)) == val {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case map[string]interface{}:
		// Check if this is a config object
		if _, ok := val["name"].(string); ok {
			name, _, _, _ := extractConfigInfo(val)
			// Return only the name, config_id will be shown as a separate field
			return name
		}
		b, _ := json.Marshal(val)
		return string(b)
	case []interface{}:
		b, _ := json.Marshal(val)
		return string(b)
	case nil:
		return "-"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}
