# gba-go

[![Tests](https://github.com/Div9851/gba-go/actions/workflows/test.yml/badge.svg)](https://github.com/Div9851/gba-go/actions/workflows/test.yml)

## Overview

A Game Boy Advance (GBA) emulator developed as a hobby project using Go and Ebitengine.

## Goal

The aim is to create an emulator capable of running Pokémon games without issues.

## Technology Stack

- **Language**: Go
- **Game Engine**: Ebitengine

## Quick Start

### Building

```bash
# Clone the repository
git clone https://github.com/Div9851/gba-go
cd gba-go

# Build the emulator
make build
# or
go build -o bin/gba-go ./cmd/gba-go

# Run the emulator
make run
# or
./bin/gba-go
```

### Project Structure

```
gba-go/
├── cmd/gba-go/          # Main application entry point
├── internal/             # Internal packages
│   ├── cpu/              # ARM7TDMI CPU emulation
│   ├── memory/           # Memory management
│   ├── ppu/              # Graphics processing
│   ├── apu/              # Audio processing
│   ├── input/            # Input handling
│   └── cartridge/        # Cartridge handling
├── pkg/emulator/         # Public emulator interface
├── assets/               # Test ROMs, BIOS files
├── docs/                 # Documentation
└── tests/                # Test files
```

## Development Status

Currently under development.

## References

- [GBATEK - GBA Technical Data](https://rust-console.github.io/gbatek-gbaonly)
- [Ebitengine Documentation](https://ebitengine.org/)
