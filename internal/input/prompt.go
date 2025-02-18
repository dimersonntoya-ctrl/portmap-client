package input

import (
    "bufio"
    "fmt"
    "strings"
)

// PromptForValue prompts the user for input with optional requirement
func PromptForValue(reader *bufio.Reader, prompt string, required bool) (string, error) {
    for {
        fmt.Printf("%s: ", prompt)
        value, err := reader.ReadString('\n')
        if err != nil {
            return "", err
        }
        value = strings.TrimSpace(value)
        if value != "" || !required {
            return value, nil
        }
        fmt.Println("This field is required")
    }
}
