# Lazyssh Project Overview

## Purpose
Lazyssh is a terminal-based, interactive SSH server manager built in Go, inspired by tools like lazydocker and k9s. It provides a clean, keyboard-driven UI for navigating, connecting to, managing, and transferring files between local machine and servers defined in ~/.ssh/config. No more remembering IP addresses or running long scp commands.

## Tech Stack
- **Language**: Go 1.24.6
- **Architecture**: Hexagonal/Clean Architecture
- **UI Framework**: tview + tcell (TUI - Terminal User Interface)
- **CLI Framework**: Cobra
- **Configuration Parser**: ssh_config (fork)
- **Logging**: Zap
- **Clipboard**: atotto/clipboard

## Key Features
- 📜 SSH config parsing and management
- ➕ Add/edit/delete server entries via TUI
- 🔍 Fuzzy search by alias, IP, or tags
- 🏷 Server tagging and filtering
- ️pinning and sorting options
- 🏓 Server ping testing
- 🔗 Port forwarding and advanced SSH options
- 🔑 SSH key management and deployment
- 📁 File transfer capabilities (planned)

## Project Structure
- `cmd/` - Application entry point
- `internal/adapters/` - UI and data adapters
- `internal/core/` - Domain logic and services
- `internal/logger/` - Logging infrastructure
- `docs/` - Documentation and screenshots

## Development System
Running on macOS (Darwin) with standard unix utilities.