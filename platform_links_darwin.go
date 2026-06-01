//go:build darwin

package main

// Wails on macOS uses UTType internally, so the framework must be linked in
// the main binary even though this project does not reference it directly.
import _ "codex-session-display/internal/platform/macoslink"
