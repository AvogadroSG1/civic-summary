#!/bin/bash
# install.sh — Install civic-summary launchd agent
#
# Installs the plist to ~/Library/LaunchAgents/ and loads it.
# Safe to re-run: unloads existing agent before reloading.

set -euo pipefail

PLIST_NAME="com.poconnor.civic-summary"
PLIST_SRC="$(cd "$(dirname "$0")" && pwd)/${PLIST_NAME}.plist"
PLIST_DST="$HOME/Library/LaunchAgents/${PLIST_NAME}.plist"
LOG_DIR="$HOME/Library/Logs/civic-summary"

# Verify source plist exists
if [ ! -f "$PLIST_SRC" ]; then
    echo "Error: plist not found at $PLIST_SRC"
    exit 1
fi

# Verify binary is installed
if [ ! -f "$HOME/go/bin/civic-summary" ]; then
    echo "Warning: civic-summary binary not found at $HOME/go/bin/civic-summary"
    echo "Run 'make install' or 'go install' first."
fi

# Create log directory
mkdir -p "$LOG_DIR"
echo "Created log directory: $LOG_DIR"

# Unload existing agent if loaded
if launchctl list "$PLIST_NAME" &>/dev/null; then
    echo "Unloading existing agent..."
    launchctl unload "$PLIST_DST" 2>/dev/null || true
fi

# Copy plist to LaunchAgents
cp "$PLIST_SRC" "$PLIST_DST"
echo "Installed plist to: $PLIST_DST"

# Load the agent
launchctl load "$PLIST_DST"
echo "Agent loaded."

# Verify
echo ""
echo "Verification:"
launchctl list | grep civic-summary || echo "Warning: agent not found in launchctl list"
echo ""
echo "Logs will be written to: $LOG_DIR"
echo "Schedule: daily at 10:00 AM"
