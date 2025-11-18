#!/bin/bash
#
# Install git hooks for Transire development
#
# This script installs the pre-commit hook that runs checks
# before allowing commits to ensure code quality.

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}Installing Transire git hooks...${NC}"
echo ""

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    echo -e "${YELLOW}Error: Not in a git repository${NC}"
    echo "Please run this script from the repository root"
    exit 1
fi

# Check if hooks directory exists
if [ ! -d ".git/hooks" ]; then
    echo -e "${YELLOW}Error: .git/hooks directory not found${NC}"
    exit 1
fi

# Install pre-commit hook
echo "Installing pre-commit hook..."
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
echo -e "${GREEN}✓ Pre-commit hook installed${NC}"
echo ""

# Check for golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${YELLOW}⚠ golangci-lint not found${NC}"
    echo "The pre-commit hook will work without it, but it's recommended for better linting."
    echo ""
    echo "To install golangci-lint:"
    echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin"
    echo ""
fi

echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   Git hooks installed successfully!   ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo "The pre-commit hook will now run automatically before each commit."
echo "To bypass it in emergencies: git commit --no-verify"
echo ""
