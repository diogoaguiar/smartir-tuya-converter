# smartir-tuya-converter

Convert [SmartIR](https://github.com/litinoveweedle/SmartIR) device code files from Broadlink (Base64) format to Tuya (Raw/MQTT) format for use with Zigbee2MQTT IR blasters.

## Why

SmartIR ships device code files in Broadlink encoding. If you use a **Tuya-based Zigbee IR blaster** (e.g., ZS06, UFO-R11) through **Zigbee2MQTT**, you need the codes in Tuya's compressed raw format. This tool does that conversion.

## Install

```bash
go install github.com/diogoaguiar/smartir-tuya-converter/cmd/smartir-tuya-converter@latest
```

Or build from source:

```bash
git clone https://github.com/diogoaguiar/smartir-tuya-converter.git
cd smartir-tuya-converter
make build
# Binary is at ./bin/smartir-tuya-converter
```

## Usage

```
smartir-tuya-converter <input.json> [output.json]
```

If `output.json` is omitted, writes to stdout.

### Examples

Convert a SmartIR Daikin device file:

```bash
smartir-tuya-converter 1109.json 1109_tuya.json
```

Pipe to stdout:

```bash
smartir-tuya-converter 1109.json > 1109_tuya.json
```

The tool:
- Reads the input SmartIR JSON
- Converts all IR codes from Broadlink Base64 to Tuya compressed format
- Updates `commandsEncoding` to `"Raw"` and `supportedController` to `"MQTT"`
- Writes the converted JSON

### Supported formats

The converter handles SmartIR files with any nesting depth:
- **3-level**: `mode -> fan -> temp -> code` (e.g., model 1109)
- **4-level**: `mode -> fan -> swing -> temp -> code` (e.g., model 1116)

## How it works

The conversion pipeline:

1. Decode Broadlink base64 to raw bytes
2. Parse Broadlink variable-length IR pulse durations
3. Convert durations from Broadlink units (~32.84µs) to microseconds
4. Pack as little-endian uint16 stream
5. Compress using Tuya's LZ-style algorithm
6. Encode as base64

Zero external dependencies — only the Go standard library.

## Use with SmartIR + Zigbee2MQTT

1. Download the SmartIR device code JSON for your AC (from the [SmartIR codes repo](https://github.com/litinoveweedle/SmartIR/tree/master/codes/climate))
2. Convert it: `smartir-tuya-converter 1109.json 1109_tuya.json`
3. Place the output in `<ha-config>/custom_components/smartir/codes/climate/`
4. Configure a SmartIR climate entity pointing to your Zigbee2MQTT IR blaster's MQTT topic

See the [SmartIR setup guide](https://github.com/diogoaguiar/hvac-manager/blob/master/docs/smartir-setup.md) for detailed instructions.

## Context

This tool was extracted from [hvac-manager](https://github.com/diogoaguiar/hvac-manager), a Go microservice for HVAC control via IR. The conversion pipeline is the reusable piece — the rest of hvac-manager's functionality (MQTT, HA integration, state management) is better served by SmartIR directly.

## License

MIT
