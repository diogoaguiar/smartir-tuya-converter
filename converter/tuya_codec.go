package converter

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
)

// Broadlink IR code format constants
const (
	// BroadlinkUnit represents the time unit used by Broadlink devices (32.84 microseconds).
	// This is calculated as 269/8192, which equals approximately 0.032836914 milliseconds.
	BroadlinkUnit = 269.0 / 8192.0

	// TuyaWindowSize is the sliding window size for Tuya LZ-style compression (8KB).
	// This is 2^13 bytes, used to find matching sequences in previous data.
	TuyaWindowSize = 1 << 13 // 8192 bytes

	// TuyaMaxMatchLength is the maximum length of a matched sequence (265 bytes).
	// Calculated as 256 + 9, this limits how far back we can reference.
	TuyaMaxMatchLength = 256 + 9 // 265
)

// parseBroadlinkDurations extracts pulse durations from a Broadlink hex string.
// The Broadlink format uses variable-length encoding:
// - 8-bit values (00-FF) represent standard durations
// - 0x00 prefix indicates a 16-bit big-endian extended duration
//
// Returns a slice of duration values in Broadlink units (not yet converted to microseconds).
func parseBroadlinkDurations(hexString string) ([]int, error) {
	if len(hexString) < 8 {
		return nil, fmt.Errorf("invalid Broadlink format: too short (min 8 hex chars)")
	}

	// Extract payload length (little-endian 16-bit at position 4-8)
	// Position 4-6: low byte, Position 6-8: high byte
	lengthHex := hexString[6:8] + hexString[4:6]
	length64, err := parseHexToInt(lengthHex)
	if err != nil {
		return nil, fmt.Errorf("invalid payload length: %w", err)
	}
	length := int(length64)

	var durations []int
	i := 8 // Start after header

	// Parse IR payload: alternating mark/space durations
	for i < length*2+8 {
		if i+2 > len(hexString) {
			break
		}

		hexValue := hexString[i : i+2]

		// Check for extended 16-bit duration (big-endian)
		if hexValue == "00" {
			if i+6 > len(hexString) {
				return nil, fmt.Errorf("truncated extended duration at position %d", i)
			}
			// Big-endian: high byte first
			hexValue = hexString[i+2:i+4] + hexString[i+4:i+6]
			i += 4
		}

		// Convert hex to integer
		val, err := parseHexToInt(hexValue)
		if err != nil {
			return nil, fmt.Errorf("invalid hex value '%s' at position %d: %w", hexValue, i, err)
		}

		durations = append(durations, int(val))
		i += 2
	}

	return durations, nil
}

// convertToMicroseconds converts Broadlink duration units to microseconds and filters.
// Broadlink uses ~32.84 microsecond units. The conversion formula is:
//
//	microseconds = ceil(duration / BroadlinkUnit)
//
// Values >= 65535 are filtered out as they cannot fit in uint16 (Tuya format limitation).
func convertToMicroseconds(durations []int) []uint16 {
	result := make([]uint16, 0, len(durations))

	for _, duration := range durations {
		// Convert to microseconds and round up
		microseconds := int(math.Ceil(float64(duration) / BroadlinkUnit))

		// Filter out values that don't fit in uint16
		if microseconds < 65535 {
			result = append(result, uint16(microseconds))
		}
	}

	return result
}

// packRawBytes converts microsecond durations to little-endian 16-bit byte stream.
// Each duration is packed as a uint16 in little-endian format, which is the format
// expected by Tuya IR blasters before compression.
func packRawBytes(microseconds []uint16) []byte {
	buf := new(bytes.Buffer)
	for _, value := range microseconds {
		// Write as little-endian uint16
		binary.Write(buf, binary.LittleEndian, value)
	}
	return buf.Bytes()
}

// compressTuya compresses raw IR timing data using Tuya's LZ-style compression.
// This implements level 2 compression: eagerly use best length-distance pair found.
//
// The compression uses two types of tokens:
// 1. Literal blocks: up to 32 bytes of raw data
// 2. Distance blocks: (length, distance) pairs referencing previous data
//
// The algorithm maintains a sliding window of previously seen data (8KB) and searches
// for the longest matching sequence. If a match >= 3 bytes is found, it emits a
// distance block; otherwise it accumulates literal data.
func compressTuya(data []byte) []byte {
	out := new(bytes.Buffer)

	blockStart := 0
	pos := 0

	for pos < len(data) {
		// Try to find the best matching sequence in the sliding window
		bestLength, bestDistance := findBestMatch(data, pos)

		// Emit distance block if we found a good match (>= 3 bytes)
		if bestLength >= 3 {
			// First emit any accumulated literal data
			emitLiteralBlocks(out, data[blockStart:pos])

			// Emit the distance block
			emitDistanceBlock(out, bestLength, bestDistance)

			pos += bestLength
			blockStart = pos
		} else {
			// No good match, advance and accumulate literal data
			pos++
		}
	}

	// Emit remaining literal data
	emitLiteralBlocks(out, data[blockStart:pos])

	return out.Bytes()
}

// findBestMatch searches the sliding window for the longest matching sequence.
// Returns (length, distance) where:
// - length: number of matching bytes (0 if no match >= 3)
// - distance: how far back the match was found (1-indexed)
//
// This implements a linear search through the window for simplicity and matches
// the Python level-2 compression behavior.
func findBestMatch(data []byte, pos int) (int, int) {
	bestLength := 0
	bestDistance := 0

	// Define window boundaries: look back up to TuyaWindowSize bytes
	windowStart := pos - TuyaWindowSize
	if windowStart < 0 {
		windowStart = 0
	}

	// Search backward through the window for matches
	for distance := 1; distance <= pos-windowStart; distance++ {
		comparePos := pos - distance
		length := 0
		maxLength := TuyaMaxMatchLength
		if pos+maxLength > len(data) {
			maxLength = len(data) - pos
		}

		// Count matching bytes
		for length < maxLength && data[pos+length] == data[comparePos+length] {
			length++
		}

		// Keep track of the best match (prefer longer matches, then closer ones)
		if length > bestLength {
			bestLength = length
			bestDistance = distance
		}
	}

	return bestLength, bestDistance
}

// emitLiteralBlocks splits data into chunks of up to 32 bytes and emits each as a literal block.
func emitLiteralBlocks(out *bytes.Buffer, data []byte) {
	for i := 0; i < len(data); i += 32 {
		end := i + 32
		if end > len(data) {
			end = len(data)
		}
		emitLiteralBlock(out, data[i:end])
	}
}

// emitLiteralBlock writes a literal block token to the output stream.
// Format: [length-1] [data bytes]
// where length-1 is a 5-bit value (0-31, representing 1-32 bytes)
func emitLiteralBlock(out *bytes.Buffer, data []byte) {
	length := len(data) - 1
	if length < 0 || length >= (1<<5) {
		panic(fmt.Sprintf("invalid literal block length: %d (data length: %d)", length, len(data)))
	}
	out.WriteByte(byte(length))
	out.Write(data)
}

// emitDistanceBlock writes a distance block token to the output stream.
// Format varies based on length:
// - If length < 9:  [length<<5 | distance>>8] [distance&0xFF]
// - If length >= 9: [7<<5 | distance>>8] [distance&0xFF] [length-7]
//
// This encoding uses:
// - 13 bits for distance (0-8191, stored as distance-1)
// - 3 bits for length in header (0-7, representing lengths 2-9)
// - Optional extra byte for lengths >= 9
func emitDistanceBlock(out *bytes.Buffer, length int, distance int) {
	// Distance is 1-indexed in the API, but stored as 0-indexed
	distance--

	if distance < 0 || distance >= TuyaWindowSize {
		panic(fmt.Sprintf("distance out of range: %d (must be 0-%d)", distance+1, TuyaWindowSize))
	}

	// Length is relative to 2 (minimum match length)
	length -= 2
	if length <= 0 {
		panic(fmt.Sprintf("length too small: %d (must be >= 2)", length+2))
	}

	var block []byte

	if length >= 7 {
		// Long match: use extended encoding
		if length >= (1 << 8) {
			panic(fmt.Sprintf("length too large: %d (max %d)", length+2, TuyaMaxMatchLength))
		}
		// Header with length=7, then distance bytes, then extra length byte
		block = []byte{
			byte(7<<5 | distance>>8),
			byte(distance & 0xFF),
			byte(length - 7),
		}
	} else {
		// Short match: encode length in header
		block = []byte{
			byte(length<<5 | distance>>8),
			byte(distance & 0xFF),
		}
	}

	out.Write(block)
}

// encodeTuyaBase64 encodes compressed Tuya data to base64.
// The output is a single line (no newlines), matching the format used in SmartIR files.
func encodeTuyaBase64(compressed []byte) string {
	return base64.StdEncoding.EncodeToString(compressed)
}

// parseHexToInt converts a hex string to an integer.
// Helper function to avoid duplicating hex parsing logic.
func parseHexToInt(hexStr string) (int64, error) {
	var val int64
	_, err := fmt.Sscanf(hexStr, "%x", &val)
	return val, err
}
