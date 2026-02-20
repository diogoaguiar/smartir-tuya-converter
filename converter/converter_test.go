package converter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConvertBroadlinkToTuya_RealData tests conversion against actual SmartIR files.
func TestConvertBroadlinkToTuya_RealData(t *testing.T) {
	testDataDir := "../testdata"
	broadlinkFile := filepath.Join(testDataDir, "1109.json")

	if _, err := os.Stat(broadlinkFile); os.IsNotExist(err) {
		t.Skip("Test data not found, skipping real data test")
		return
	}

	broadlinkData, err := os.ReadFile(broadlinkFile)
	if err != nil {
		t.Fatalf("Failed to read Broadlink file: %v", err)
	}

	var broadlinkJSON map[string]interface{}
	if err := json.Unmarshal(broadlinkData, &broadlinkJSON); err != nil {
		t.Fatalf("Failed to parse Broadlink JSON: %v", err)
	}

	broadlinkCommands := broadlinkJSON["commands"].(map[string]interface{})

	// Test "off" command
	if broadlinkOff, ok := broadlinkCommands["off"].(string); ok {
		convertedOff, err := ConvertBroadlinkToTuya(broadlinkOff)
		if err != nil {
			t.Errorf("Failed to convert 'off' command: %v", err)
		}

		if convertedOff == "" {
			t.Error("Converted 'off' command is empty")
		}

		if convertedOff == broadlinkOff {
			t.Error("'off' command was not converted (same as input)")
		}

		t.Logf("Successfully converted 'off' command: %d bytes", len(convertedOff))
	}

	// Test a sample of mode-based commands
	testCases := []struct {
		mode string
		fan  string
		temp string
	}{
		{"cool", "low", "21"},
		{"heat", "auto", "25"},
		{"dry", "low", "20"},
		{"fan_only", "high", "25"},
	}

	successCount := 0
	for _, tc := range testCases {
		if modeData, ok := broadlinkCommands[tc.mode].(map[string]interface{}); ok {
			if fanData, ok := modeData[tc.fan].(map[string]interface{}); ok {
				if broadlinkCode, ok := fanData[tc.temp].(string); ok {
					convertedCode, err := ConvertBroadlinkToTuya(broadlinkCode)
					if err != nil {
						t.Errorf("Failed to convert code for mode=%s fan=%s temp=%s: %v",
							tc.mode, tc.fan, tc.temp, err)
						continue
					}

					if convertedCode == "" {
						t.Errorf("Empty result for mode=%s fan=%s temp=%s", tc.mode, tc.fan, tc.temp)
						continue
					}

					if convertedCode == broadlinkCode {
						t.Errorf("Code was not converted for mode=%s fan=%s temp=%s", tc.mode, tc.fan, tc.temp)
						continue
					}

					successCount++
					t.Logf("Converted mode=%s fan=%s temp=%s: %d bytes", tc.mode, tc.fan, tc.temp, len(convertedCode))
				}
			}
		}
	}

	if successCount == 0 {
		t.Error("No codes were successfully converted")
	} else {
		t.Logf("Successfully converted %d/%d test codes", successCount, len(testCases))
	}
}

// TestConvertBroadlinkToTuya_EdgeCases tests error handling and edge cases.
func TestConvertBroadlinkToTuya_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorText   string
	}{
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
			errorText:   "empty",
		},
		{
			name:        "Invalid base64",
			input:       "Not!Valid@Base64",
			expectError: true,
			errorText:   "invalid base64",
		},
		{
			name:        "Too short",
			input:       "JgA=",
			expectError: true,
			errorText:   "too short",
		},
		{
			name:        "Whitespace handling",
			input:       "  JgBsAaVGDDoMFw4WDBcPOAwXDhYOFQ4WDTkNFww7DDoNFw06DDoNOg06DToMFw45DRcNFg4VDhYOFQ0XDTkNOg0XDRYOFQ4WDhUOOQ0WDRcOFQ4WDxQPFQwXDhUOFg4VDhYNFg4VDRcMOw05DToNOg0WDxUPFA4AA8SmRQ06DBcPFQ0WDjkMFw4WDBcOFgw6DRcNOgw6DRcOOQw6DToNOg06DRYPOA0WDRcNFg4WDBcNFw44DToNFw0WDhUNFw8UDxUNFg0WDxUOFQ4WDDoNOg0XDhUOFg0WDToNFg4WDRYPFA4WDRYNFw4VDxQOFg4VDhYMFw4VDhYOFQ4WDBgNFg4VDhUOFgwXDhYMFw4VDxUOFQ4WDRYNFg4WDhUNFw0WDhUPFQw7DBcNFwwXDhUOOQ45DRYPOA0XDRYOFQ4WDhUOFg0WDhYNFg4VDxUNFg4VDhYOFQ4WDToMFw4VDjkNOg0WDRcOFQ45DRYOOQ0ADQUAAAAAAAAAAAAA  ",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertBroadlinkToTuya(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorText)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorText)) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if result == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

// TestConvertCommands_Recursive tests recursive conversion of nested command structures.
func TestConvertCommands_Recursive(t *testing.T) {
	testDataDir := "../testdata"

	// Test with 1116.json which has 4-level nesting (mode -> fan -> swing -> temp -> code)
	file1116 := filepath.Join(testDataDir, "1116.json")
	if _, err := os.Stat(file1116); os.IsNotExist(err) {
		t.Skip("Test data not found, skipping recursive conversion test")
		return
	}

	data, err := os.ReadFile(file1116)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	commands := jsonData["commands"].(map[string]interface{})

	// Convert all commands recursively
	converted, err := ConvertCommands(commands)
	if err != nil {
		t.Fatalf("Failed to convert commands: %v", err)
	}

	// Verify the "off" command was converted
	if offCode, ok := converted["off"].(string); ok {
		if offCode == commands["off"].(string) {
			t.Error("'off' command was not converted")
		}
	} else {
		t.Error("'off' command missing or not a string")
	}

	// Verify the 4-level nested structure: cool -> level1 -> off -> 16
	cool, ok := converted["cool"].(map[string]interface{})
	if !ok {
		t.Fatal("'cool' mode missing or not a map")
	}

	level1, ok := cool["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("'level1' fan missing or not a map")
	}

	swingOff, ok := level1["off"].(map[string]interface{})
	if !ok {
		t.Fatal("'off' swing missing or not a map")
	}

	code16, ok := swingOff["16"].(string)
	if !ok {
		t.Fatal("temp '16' missing or not a string")
	}

	if code16 == "" {
		t.Error("Converted code for cool/level1/off/16 is empty")
	}

	// Get original for comparison
	origCool := commands["cool"].(map[string]interface{})
	origLevel1 := origCool["level1"].(map[string]interface{})
	origSwingOff := origLevel1["off"].(map[string]interface{})
	origCode16 := origSwingOff["16"].(string)

	if code16 == origCode16 {
		t.Error("Code was not converted (same as input)")
	}

	t.Logf("Successfully converted 4-level nested code: %d bytes", len(code16))
}

// TestConvertBroadlinkToTuya_GoldenReference tests that our Go converter output
// decompresses to the same raw IR data as the Python reference converter.
// The compressed bytes may differ (different match-finding strategies), but the
// decompressed raw data must be identical — this is what the IR blaster transmits.
func TestConvertBroadlinkToTuya_GoldenReference(t *testing.T) {
	testFiles := []struct {
		broadlink string
		reference string
	}{
		{"../testdata/1109.json", "../testdata/1109_tuya_reference.json"},
		{"../testdata/1116.json", "../testdata/1116_tuya_reference.json"},
	}

	for _, tf := range testFiles {
		t.Run(tf.broadlink, func(t *testing.T) {
			if _, err := os.Stat(tf.broadlink); os.IsNotExist(err) {
				t.Skipf("Test data not found: %s", tf.broadlink)
			}
			if _, err := os.Stat(tf.reference); os.IsNotExist(err) {
				t.Skipf("Reference data not found: %s", tf.reference)
			}

			// Load Broadlink source
			srcData, err := os.ReadFile(tf.broadlink)
			if err != nil {
				t.Fatalf("Failed to read source: %v", err)
			}
			var srcJSON map[string]interface{}
			if err := json.Unmarshal(srcData, &srcJSON); err != nil {
				t.Fatalf("Failed to parse source JSON: %v", err)
			}

			// Load Python reference
			refData, err := os.ReadFile(tf.reference)
			if err != nil {
				t.Fatalf("Failed to read reference: %v", err)
			}
			var refJSON map[string]interface{}
			if err := json.Unmarshal(refData, &refJSON); err != nil {
				t.Fatalf("Failed to parse reference JSON: %v", err)
			}

			// Convert with our Go converter
			srcCommands := srcJSON["commands"].(map[string]interface{})
			goConverted, err := ConvertCommands(srcCommands)
			if err != nil {
				t.Fatalf("Go conversion failed: %v", err)
			}

			refCommands := refJSON["commands"].(map[string]interface{})

			// Recursively compare all codes
			var compared, matched int
			var compareCommands func(goMap, refMap map[string]interface{}, path string)
			compareCommands = func(goMap, refMap map[string]interface{}, path string) {
				for key, goVal := range goMap {
					refVal, ok := refMap[key]
					if !ok {
						t.Errorf("Key %s/%s missing from reference", path, key)
						continue
					}

					switch gv := goVal.(type) {
					case string:
						rv, ok := refVal.(string)
						if !ok {
							t.Errorf("Type mismatch at %s/%s", path, key)
							continue
						}

						compared++

						// Decompress both and compare raw data
						goCompressed, err := base64.StdEncoding.DecodeString(gv)
						if err != nil {
							t.Errorf("Failed to decode Go base64 at %s/%s: %v", path, key, err)
							continue
						}
						refCompressed, err := base64.StdEncoding.DecodeString(rv)
						if err != nil {
							t.Errorf("Failed to decode ref base64 at %s/%s: %v", path, key, err)
							continue
						}

						goRaw := decompressTuya(goCompressed)
						refRaw := decompressTuya(refCompressed)

						if !bytes.Equal(goRaw, refRaw) {
							t.Errorf("Raw IR data mismatch at %s/%s: Go=%d bytes, Ref=%d bytes",
								path, key, len(goRaw), len(refRaw))
						} else {
							matched++
						}

					case map[string]interface{}:
						rv, ok := refVal.(map[string]interface{})
						if !ok {
							t.Errorf("Type mismatch at %s/%s", path, key)
							continue
						}
						compareCommands(gv, rv, path+"/"+key)
					}
				}
			}

			compareCommands(goConverted, refCommands, "")
			t.Logf("Compared %d codes, %d matched (%s)", compared, matched, tf.broadlink)

			if compared == 0 {
				t.Error("No codes were compared")
			}
			if matched != compared {
				t.Errorf("%d/%d codes had mismatched raw IR data", compared-matched, compared)
			}
		})
	}
}

// BenchmarkConvertBroadlinkToTuya benchmarks the conversion performance.
func BenchmarkConvertBroadlinkToTuya(b *testing.B) {
	testDataDir := "../testdata"
	broadlinkFile := filepath.Join(testDataDir, "1109.json")

	if _, err := os.Stat(broadlinkFile); os.IsNotExist(err) {
		b.Skip("Test data not found")
		return
	}

	data, err := os.ReadFile(broadlinkFile)
	if err != nil {
		b.Fatalf("Failed to read test file: %v", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		b.Fatalf("Failed to parse JSON: %v", err)
	}

	commands := jsonData["commands"].(map[string]interface{})
	sampleCode := commands["off"].(string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ConvertBroadlinkToTuya(sampleCode)
		if err != nil {
			b.Fatalf("Conversion failed: %v", err)
		}
	}
}
