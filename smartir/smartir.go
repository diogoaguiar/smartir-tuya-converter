package smartir

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/diogoaguiar/smartir-tuya-converter/converter"
)

// File represents the structure of a SmartIR JSON device code file.
type File struct {
	Manufacturer        string                 `json:"manufacturer"`
	SupportedModels     []string               `json:"supportedModels"`
	CommandsEncoding    string                 `json:"commandsEncoding"`
	SupportedController string                 `json:"supportedController"`
	MinTemperature      int                    `json:"minTemperature"`
	MaxTemperature      int                    `json:"maxTemperature"`
	Precision           json.Number            `json:"precision"`
	OperationModes      []string               `json:"operationModes"`
	FanModes            []string               `json:"fanModes"`
	SwingModes          []string               `json:"swingModes,omitempty"`
	Commands            map[string]interface{} `json:"commands"`
}

// ReadFile reads and parses a SmartIR JSON file.
func ReadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &f, nil
}

// IsBroadlink returns true if the file uses Broadlink encoding (needs conversion).
func (f *File) IsBroadlink() bool {
	return f.CommandsEncoding == "Base64"
}

// IsRaw returns true if the file already uses Raw/Tuya encoding.
func (f *File) IsRaw() bool {
	return f.CommandsEncoding == "Raw"
}

// ConvertToTuya converts all Broadlink IR codes in the file to Tuya format.
// Updates the commands in-place and sets the metadata to reflect Tuya format.
// Returns an error if conversion fails or if the file is not in Broadlink format.
func (f *File) ConvertToTuya() error {
	if !f.IsBroadlink() {
		return fmt.Errorf("file is not in Broadlink format (commandsEncoding: %s)", f.CommandsEncoding)
	}

	converted, err := converter.ConvertCommands(f.Commands)
	if err != nil {
		return fmt.Errorf("failed to convert commands: %w", err)
	}

	f.Commands = converted
	f.CommandsEncoding = "Raw"
	f.SupportedController = "MQTT"

	return nil
}

// WriteJSON writes the SmartIR file as pretty-printed JSON to the given path.
func (f *File) WriteJSON(path string) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

