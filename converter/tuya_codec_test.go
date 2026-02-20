package converter

import (
	"bytes"
	"testing"
)

// decompressTuya decompresses Tuya LZ-style compressed data.
// Used only for testing to verify compressed output is correct.
func decompressTuya(data []byte) []byte {
	var out []byte
	pos := 0

	for pos < len(data) {
		b := data[pos]
		lengthBits := (b >> 5) & 0x07

		if lengthBits == 0 {
			// Literal block
			litLen := int(b&0x1f) + 1
			pos++
			out = append(out, data[pos:pos+litLen]...)
			pos += litLen
		} else {
			// Distance block
			distHigh := int(b & 0x1f)
			pos++

			length := int(lengthBits) + 2
			if lengthBits == 7 {
				extra := int(data[pos])
				pos++
				length = 7 + 2 + extra
			}

			distLow := int(data[pos])
			pos++
			distance := (distHigh<<8 | distLow) + 1

			for i := 0; i < length; i++ {
				out = append(out, out[len(out)-distance])
			}
		}
	}

	return out
}

// TestCompressDecompressRoundtrip verifies that compress → decompress produces
// the original data.
func TestCompressDecompressRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"short", []byte("hello world")},
		{"repeated", []byte("abcabcabcabcabcabcabcabc")},
		{"binary", func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed := compressTuya(tt.data)
			decompressed := decompressTuya(compressed)

			if !bytes.Equal(decompressed, tt.data) {
				t.Fatalf("roundtrip mismatch: got %d bytes, want %d bytes", len(decompressed), len(tt.data))
			}
		})
	}
}

// TestEmitDistanceBlockFormat verifies the distance block byte ordering matches
// the Tuya specification: [header] [extra_length (if long)] [distance_low].
func TestEmitDistanceBlockFormat(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		distance int
		want     []byte
	}{
		{
			name:     "short match (length=3, distance=1)",
			length:   3,
			distance: 1,
			// length-2=1, distance-1=0: [1<<5|0>>8, 0&0xFF] = [0x20, 0x00]
			want: []byte{0x20, 0x00},
		},
		{
			name:     "short match (length=8, distance=5)",
			length:   8,
			distance: 5,
			// length-2=6, distance-1=4: [6<<5|4>>8, 4&0xFF] = [0xC0, 0x04]
			want: []byte{0xC0, 0x04},
		},
		{
			name:     "long match (length=9, distance=1)",
			length:   9,
			distance: 1,
			// length-2=7 (>=7): [7<<5|0>>8, 7-7, 0&0xFF] = [0xE0, 0x00, 0x00]
			want: []byte{0xE0, 0x00, 0x00},
		},
		{
			name:     "long match (length=20, distance=100)",
			length:   20,
			distance: 100,
			// length-2=18 (>=7): [7<<5|99>>8, 18-7, 99&0xFF] = [0xE0, 0x0B, 0x63]
			want: []byte{0xE0, 0x0B, 0x63},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			emitDistanceBlock(out, tt.length, tt.distance)
			got := out.Bytes()

			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
