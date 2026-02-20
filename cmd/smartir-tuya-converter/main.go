package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/diogoaguiar/smartir-tuya-converter/smartir"
)

const usage = `Usage: smartir-tuya-converter <input.json> [output.json]

  Converts a SmartIR device code file from Broadlink (Base64) format to
  Tuya (Raw/MQTT) format for use with Zigbee2MQTT IR blasters.

  If output.json is omitted, writes to stdout.

Examples:
  smartir-tuya-converter 1109.json 1109_tuya.json
  smartir-tuya-converter 1109.json > 1109_tuya.json`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintln(os.Stderr, usage)
		if len(os.Args) >= 2 {
			os.Exit(0)
		}
		os.Exit(1)
	}

	inputPath := os.Args[1]
	var outputPath string
	if len(os.Args) >= 3 {
		outputPath = os.Args[2]
	}

	// Read input file
	f, err := smartir.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Check if already converted
	if f.IsRaw() {
		fmt.Fprintf(os.Stderr, "Warning: file is already in Raw/Tuya format, no conversion needed.\n")
		os.Exit(0)
	}

	// Check for supported format
	if !f.IsBroadlink() {
		fmt.Fprintf(os.Stderr, "Error: unsupported commandsEncoding %q (expected \"Base64\")\n", f.CommandsEncoding())
		os.Exit(1)
	}

	// Convert
	if err := f.ConvertToTuya(); err != nil {
		fmt.Fprintf(os.Stderr, "Error converting: %v\n", err)
		os.Exit(1)
	}

	// Output
	if outputPath != "" {
		if err := f.WriteJSON(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Converted %s -> %s\n", inputPath, outputPath)
	} else {
		// Write to stdout
		data, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	}
}
