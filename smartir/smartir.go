package smartir

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/diogoaguiar/smartir-tuya-converter/converter"
)

// File represents a SmartIR JSON device code file.
// Uses a raw map internally to preserve all fields, including unknown ones.
type File struct {
	raw map[string]interface{}
}

// ReadFile reads and parses a SmartIR JSON file.
func ReadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &File{raw: raw}, nil
}

// CommandsEncoding returns the commandsEncoding field value.
func (f *File) CommandsEncoding() string {
	if v, ok := f.raw["commandsEncoding"].(string); ok {
		return v
	}
	return ""
}

// IsBroadlink returns true if the file uses Broadlink encoding (needs conversion).
func (f *File) IsBroadlink() bool {
	return f.CommandsEncoding() == "Base64"
}

// IsRaw returns true if the file already uses Raw/Tuya encoding.
func (f *File) IsRaw() bool {
	return f.CommandsEncoding() == "Raw"
}

// ConvertToTuya converts all Broadlink IR codes in the file to Tuya format.
// Preserves all fields in the JSON, only modifying commands, commandsEncoding,
// and supportedController.
func (f *File) ConvertToTuya() error {
	if !f.IsBroadlink() {
		return fmt.Errorf("file is not in Broadlink format (commandsEncoding: %s)", f.CommandsEncoding())
	}

	commands, ok := f.raw["commands"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid 'commands' field")
	}

	converted, err := converter.ConvertCommands(commands)
	if err != nil {
		return fmt.Errorf("failed to convert commands: %w", err)
	}

	f.raw["commands"] = converted
	f.raw["commandsEncoding"] = "Raw"
	f.raw["supportedController"] = "MQTT"

	return nil
}

// WriteJSON writes the SmartIR file as pretty-printed JSON to the given path.
func (f *File) WriteJSON(path string) error {
	data, err := json.MarshalIndent(f.raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// MarshalJSON returns the JSON representation of the file.
func (f *File) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.raw)
}
