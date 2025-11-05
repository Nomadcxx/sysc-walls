#!/usr/bin/env bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ASCII art
echo -e "${CYAN}"
cat << "EOF"
▄▀▀▀▀ █   █ ▄▀▀▀▀ ▄▀▀▀▀          ▄▀ █   █
 ▀▀▀▄ ▀▀▀▀█  ▀▀▀▄ █     ▀▀▀▀▀  ▄▀   █ █ █
▀▀▀▀  ▀▀▀▀▀ ▀▀▀▀   ▀▀▀▀       ▀      ▀ ▀
EOF
echo -e "${NC}"
echo -e "${CYAN}Terminal Screensaver Installer${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (use sudo)${NC}"
    exit 1
fi

# Check for required commands
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed${NC}"
        echo -e "${YELLOW}Install it with your package manager and try again${NC}"
        exit 1
    fi
}

echo -e "${CYAN}Checking dependencies...${NC}"
check_command git
check_command go

# Get the actual user (when running with sudo)
ACTUAL_USER="${SUDO_USER:-$USER}"
ACTUAL_HOME=$(eval echo "~$ACTUAL_USER")

# Create temporary directory
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

echo -e "${GREEN}✓${NC} Dependencies OK"
echo ""

# Clone repository
echo -e "${CYAN}Cloning sysc-walls...${NC}"
git clone --quiet https://github.com/Nomadcxx/sysc-walls.git
cd sysc-walls
echo -e "${GREEN}✓${NC} Repository cloned"

# Clone sysc-Go
echo -e "${CYAN}Cloning sysc-Go animation library...${NC}"
git clone --quiet https://github.com/Nomadcxx/sysc-Go.git
echo -e "${GREEN}✓${NC} Animation library cloned"

# Build binaries
echo -e "${CYAN}Building binaries...${NC}"
go build -o daemon ./cmd/daemon/ 2>&1 | grep -v "^#" || true
go build -o display ./cmd/display/ 2>&1 | grep -v "^#" || true
go build -o client ./cmd/client/ 2>&1 | grep -v "^#" || true
echo -e "${GREEN}✓${NC} Binaries built"

# Install binaries
echo -e "${CYAN}Installing to /usr/local/bin...${NC}"
cp daemon /usr/local/bin/sysc-walls-daemon
cp display /usr/local/bin/sysc-walls-display
cp client /usr/local/bin/sysc-walls-client
chmod +x /usr/local/bin/sysc-walls-daemon
chmod +x /usr/local/bin/sysc-walls-display
chmod +x /usr/local/bin/sysc-walls-client
echo -e "${GREEN}✓${NC} Binaries installed"

# Install systemd service
echo -e "${CYAN}Installing systemd service...${NC}"
SYSTEMD_DIR="$ACTUAL_HOME/.config/systemd/user"
sudo -u "$ACTUAL_USER" mkdir -p "$SYSTEMD_DIR"
cp systemd/sysc-walls-user.service "$SYSTEMD_DIR/sysc-walls.service"
chown "$ACTUAL_USER:$ACTUAL_USER" "$SYSTEMD_DIR/sysc-walls.service"
echo -e "${GREEN}✓${NC} Systemd service installed"

# Cleanup
cd /
rm -rf "$TEMP_DIR"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Installation complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${CYAN}Test your installation:${NC}"
echo -e "  ${YELLOW}sysc-walls-daemon -test${NC}              # Quick test"
echo -e "  ${YELLOW}sysc-walls-daemon -test -debug${NC}       # Test with diagnostics"
echo ""
echo -e "${CYAN}Enable and start the service:${NC}"
echo -e "  ${YELLOW}systemctl --user daemon-reload${NC}"
echo -e "  ${YELLOW}systemctl --user enable sysc-walls.service${NC}"
echo -e "  ${YELLOW}systemctl --user start sysc-walls.service${NC}"
echo ""
echo -e "${CYAN}Configuration:${NC}"
echo -e "  ${YELLOW}~/.config/sysc-walls/daemon.conf${NC}"
echo ""
