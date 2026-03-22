#!/bin/bash
set -e

SKILL_DIR="$HOME/.claude/skills"
SKILL_URL="https://raw.githubusercontent.com/sderosiaux/ticker-cli/main/skills/ticker.md"
DEST="$SKILL_DIR/ticker.md"

mkdir -p "$SKILL_DIR"

echo "Downloading ticker skill..."
curl -sSL "$SKILL_URL" -o "$DEST"

echo "Installed to $DEST"
echo ""
echo "The skill triggers automatically when you mention stock prices,"
echo "crypto, forex, or commodities in Claude Code conversations."
