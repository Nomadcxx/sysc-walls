#!/bin/bash
# sync-sysc-go.sh - Sync sysc-Go animations from main repository
# This ensures sysc-walls uses the latest animation code

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYSC_WALLS_ROOT="$(dirname "$SCRIPT_DIR")"

# Source and destination paths
SYSC_GO_MAIN="/home/nomadx/Documents/sysc-Go"
SYSC_GO_LOCAL="$SYSC_WALLS_ROOT/sysc-Go"

# Check if main sysc-Go repo exists
if [ ! -d "$SYSC_GO_MAIN" ]; then
    echo -e "${RED}Error: Main sysc-Go repository not found at: $SYSC_GO_MAIN${NC}"
    echo "Please clone sysc-Go or update the path in this script."
    exit 1
fi

# Check if local sysc-Go directory exists
if [ ! -d "$SYSC_GO_LOCAL" ]; then
    echo -e "${RED}Error: Local sysc-Go directory not found at: $SYSC_GO_LOCAL${NC}"
    exit 1
fi

echo -e "${BLUE}╔═══════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   sysc-Go Animation Synchronization      ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════╝${NC}"
echo ""

# Check for uncommitted changes in main repo
echo -e "${YELLOW}→${NC} Checking main sysc-Go repository status..."
cd "$SYSC_GO_MAIN"
if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    echo -e "${YELLOW}  Warning: Main sysc-Go repository has uncommitted changes${NC}"
    echo -e "${YELLOW}  Consider committing changes before syncing${NC}"
    read -p "  Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Sync cancelled${NC}"
        exit 1
    fi
fi

# Get main repo version
MAIN_VERSION=$(grep 'const version = ' "$SYSC_GO_MAIN/cmd/syscgo/main.go" | cut -d'"' -f2)
echo -e "${GREEN}✓${NC} Main sysc-Go version: ${GREEN}$MAIN_VERSION${NC}"

# Get current local version
if [ -f "$SYSC_GO_LOCAL/cmd/syscgo/main.go" ]; then
    LOCAL_VERSION=$(grep 'const version = ' "$SYSC_GO_LOCAL/cmd/syscgo/main.go" | cut -d'"' -f2)
    echo -e "${BLUE}  Local sysc-Go version: $LOCAL_VERSION${NC}"
else
    LOCAL_VERSION="unknown"
    echo -e "${YELLOW}  Local sysc-Go version: unknown${NC}"
fi

echo ""
echo -e "${YELLOW}→${NC} Syncing animation files..."

# Sync animations directory
SYNCED_COUNT=0
FAILED_COUNT=0

for file in "$SYSC_GO_MAIN/animations"/*.go; do
    filename=$(basename "$file")
    if cp "$file" "$SYSC_GO_LOCAL/animations/$filename" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} Synced: $filename"
        ((SYNCED_COUNT++))
    else
        echo -e "  ${RED}✗${NC} Failed: $filename"
        ((FAILED_COUNT++))
    fi
done

echo ""
echo -e "${YELLOW}→${NC} Syncing additional files..."

# Sync go.mod
if [ -f "$SYSC_GO_MAIN/go.mod" ]; then
    cp "$SYSC_GO_MAIN/go.mod" "$SYSC_GO_LOCAL/" 2>/dev/null && \
        echo -e "  ${GREEN}✓${NC} Synced: go.mod" || \
        echo -e "  ${YELLOW}⚠${NC} Skipped: go.mod"
fi

# Sync go.sum
if [ -f "$SYSC_GO_MAIN/go.sum" ]; then
    cp "$SYSC_GO_MAIN/go.sum" "$SYSC_GO_LOCAL/" 2>/dev/null && \
        echo -e "  ${GREEN}✓${NC} Synced: go.sum" || \
        echo -e "  ${YELLOW}⚠${NC} Skipped: go.sum"
fi

# Sync cmd directory (for version info)
if [ -d "$SYSC_GO_MAIN/cmd" ]; then
    rsync -a --exclude='*.exe' "$SYSC_GO_MAIN/cmd/" "$SYSC_GO_LOCAL/cmd/" 2>/dev/null && \
        echo -e "  ${GREEN}✓${NC} Synced: cmd/" || \
        echo -e "  ${YELLOW}⚠${NC} Skipped: cmd/"
fi

# Sync tui directory
if [ -d "$SYSC_GO_MAIN/tui" ]; then
    rsync -a --exclude='*.exe' "$SYSC_GO_MAIN/tui/" "$SYSC_GO_LOCAL/tui/" 2>/dev/null && \
        echo -e "  ${GREEN}✓${NC} Synced: tui/" || \
        echo -e "  ${YELLOW}⚠${NC} Skipped: tui/"
fi

echo ""
echo -e "${BLUE}╔═══════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Sync Summary                            ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════╝${NC}"
echo -e "  ${GREEN}✓${NC} Synced: $SYNCED_COUNT animation files"
if [ $FAILED_COUNT -gt 0 ]; then
    echo -e "  ${RED}✗${NC} Failed: $FAILED_COUNT files"
fi
echo -e "  ${BLUE}Version: $MAIN_VERSION${NC}"
echo ""

# Update sysc-walls MinimumSyscGoVersion if needed
CONFIG_FILE="$SYSC_WALLS_ROOT/internal/config/config.go"
if [ -f "$CONFIG_FILE" ]; then
    CURRENT_MIN=$(grep 'MinimumSyscGoVersion = ' "$CONFIG_FILE" | cut -d'"' -f2)
    if [ "$CURRENT_MIN" != "$MAIN_VERSION" ]; then
        echo -e "${YELLOW}→${NC} Updating MinimumSyscGoVersion..."
        sed -i "s/MinimumSyscGoVersion = \".*\"/MinimumSyscGoVersion = \"$MAIN_VERSION\"/" "$CONFIG_FILE"
        echo -e "  ${GREEN}✓${NC} Updated: $CURRENT_MIN → $MAIN_VERSION"
    else
        echo -e "  ${GREEN}✓${NC} MinimumSyscGoVersion already up to date"
    fi
fi

echo ""
echo -e "${GREEN}✓ Sync complete!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Review changes: ${BLUE}git diff${NC}"
echo -e "  2. Rebuild: ${BLUE}make build${NC}"
echo -e "  3. Test effects: ${BLUE}make test${NC}"
echo ""
