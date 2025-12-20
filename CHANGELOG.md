# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.2] - 2025-12-11

### Features

- **Action Management System**: Create, organize, and execute custom actions.
  - Support for nested Action Groups (folders).
  - Reorder actions and groups via move up/down buttons.
- **Scripting Support**:
  - **AppleScript**: Execution and syntax checking on macOS.
  - **Shell Commands**: Bash/Zsh on Unix, PowerShell on Windows.
  - **Syntax Highlighting**: Real-time syntax highlighting for code editors.
- **Inter-App Communication**:
  - New **Message Mapping** tab to map incoming MIDI messages (Note, CC, Program Change) to actions.
  - Support for **Generic MIDI Devices** to receive messages from other software/hardware without using the button grid layout.
- **UI Enhancements**:
  - Refined window layout with dedicated tabs for Devices, Menu Editor, Actions, and Message Mapping.
  - Improved MIDI device configuration with type selection (Classic, Colorful, Generic).

### Refactoring

- Major refactoring of the main window code into modular tab components (`devices_tab.go`, `menu_editor_tab.go`, `actions_tab.go`, `mapping_tab.go`).
- Modularized MIDI device support using a new `Device` interface, making it easier to add support for new controllers.
- Modularized Action execution using `ActionHandler` interface, separating AppleScript and Shell Command logic.

## [0.0.1] - 2025-12-09

### Features

- Initial release with Proof of Concept functionality
- MIDI device detection and management
- Layout configuration and persistence
- System tray integration for background operation
- Launch at startup
- macOS support with Dock hiding (agent app mode)
