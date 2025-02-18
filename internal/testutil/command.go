package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"portmap.io/client/internal/output"
)

func ExecuteCommand(root *cobra.Command, args ...string) (map[string]interface{}, error) {
	// Create a buffer for output
	buf := new(bytes.Buffer)

	// Save original writer and restore it after the test
	oldWriter := output.GetWriter()
	defer output.SetWriter(oldWriter)

	// Set buffer as writer
	output.SetWriter(buf)

	// Set command arguments
	root.SetArgs(args)

	// Execute command
	err := root.Execute()
	if err != nil {
		return nil, err
	}

	// Parse JSON output
	jsonStr := strings.TrimSpace(buf.String())
	if jsonStr == "" {
		return nil, fmt.Errorf("no output from command")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %v\nOutput: %s", err, jsonStr)
	}

	return result, nil
}
