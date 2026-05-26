// Package core owns Portside's shared state model, runner adapters, profile
// management, logs, snapshots, and command execution primitives.
//
// CLI and TUI call this package directly. The macOS app should call the
// bundled portside helper and consume its JSON or NDJSON protocol.
package core
