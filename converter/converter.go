package converter

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// ConvertBroadlinkToTuya converts a Broadlink IR code to Tuya compressed format.
// This function orchestrates the complete conversion pipeline:
//  1. Decode Broadlink base64 to hex
//  2. Parse Broadlink durations from hex payload
//  3. Convert durations to microseconds
//  4. Pack as raw bytes (little-endian uint16 stream)
//  5. Compress using Tuya LZ-style algorithm
//  6. Encode result as base64
//
// Input:  Broadlink IR code (e.g., "JgBsAaVGDDoMFw4W...")
// Output: Tuya IR code (e.g., "D6ETVAhuAecGbgG9...")
//
// Returns an error if the input is invalid or conversion fails.
func ConvertBroadlinkToTuya(broadlinkCode string) (string, error) {
	// Validate input
	broadlinkCode = strings.TrimSpace(broadlinkCode)
	if broadlinkCode == "" {
		return "", fmt.Errorf("empty Broadlink code")
	}

	// Step 1: Decode base64 to get hex string
	decoded, err := base64.StdEncoding.DecodeString(broadlinkCode)
	if err != nil {
		return "", fmt.Errorf("invalid base64 encoding: %w", err)
	}

	// Convert bytes to hex string
	hexString := fmt.Sprintf("%x", decoded)

	// Step 2: Parse Broadlink durations
	durations, err := parseBroadlinkDurations(hexString)
	if err != nil {
		return "", fmt.Errorf("failed to parse Broadlink format: %w", err)
	}

	if len(durations) == 0 {
		return "", fmt.Errorf("no IR durations found in Broadlink code")
	}

	// Step 3: Convert to microseconds and filter
	microseconds := convertToMicroseconds(durations)
	if len(microseconds) == 0 {
		return "", fmt.Errorf("all durations filtered out (too large for uint16)")
	}

	// Step 4: Pack as raw bytes (little-endian uint16 stream)
	rawBytes := packRawBytes(microseconds)

	// Step 5: Compress using Tuya algorithm
	compressed := compressTuya(rawBytes)

	// Step 6: Encode as base64
	tuyaCode := encodeTuyaBase64(compressed)

	return tuyaCode, nil
}

// ConvertCommands recursively converts all Broadlink IR codes in a commands structure
// to Tuya format. This handles the nested map structure used in SmartIR files:
//
//	commands: {
//	  off: "JgB...",
//	  cool: {
//	    auto: {
//	      "18": "JgB...",
//	      "19": "JgB..."
//	    }
//	  }
//	}
//
// The function preserves the structure while converting only string values (IR codes).
func ConvertCommands(commands map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range commands {
		switch v := value.(type) {
		case string:
			// Convert IR code string
			converted, err := ConvertBroadlinkToTuya(v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert code for key '%s': %w", key, err)
			}
			result[key] = converted

		case map[string]interface{}:
			// Recursively convert nested maps
			converted, err := ConvertCommands(v)
			if err != nil {
				return nil, err
			}
			result[key] = converted

		default:
			// Preserve other value types as-is (e.g., null, numbers)
			result[key] = v
		}
	}

	return result, nil
}
