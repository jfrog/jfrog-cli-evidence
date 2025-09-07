#!/bin/bash
# Setup script for Git hooks in JFrog CLI Evidence project
# This script installs or updates the git hooks for the project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the repository root directory
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"

if [ -z "$REPO_ROOT" ]; then
    echo -e "${RED}Error: Not in a git repository${NC}"
    exit 1
fi

HOOKS_DIR="$REPO_ROOT/.git/hooks"
SCRIPTS_DIR="$REPO_ROOT/scripts"

echo -e "${GREEN}Setting up Git hooks for JFrog CLI Evidence...${NC}"

# Function to create a hook
create_hook() {
    local hook_name=$1
    local hook_file="$HOOKS_DIR/$hook_name"
    
    # Check if hook already exists
    if [ -f "$hook_file" ] && [ ! -L "$hook_file" ]; then
        echo -e "${YELLOW}Warning: $hook_name hook already exists. Creating backup...${NC}"
        mv "$hook_file" "$hook_file.backup.$(date +%Y%m%d_%H%M%S)"
    fi
    
    # Create the hook based on the name
    case "$hook_name" in
        "pre-commit")
            cat > "$hook_file" << 'EOF'
#!/bin/sh
# Pre-commit hook for JFrog CLI Evidence
# This hook runs formatting, vetting, and short tests before each commit

set -e

echo "Running pre-commit checks..."

# Change to the repository root directory
cd "$(git rev-parse --show-toplevel)"

# Run the pre-commit target from Makefile
make pre-commit

if [ $? -ne 0 ]; then
    echo "❌ Pre-commit checks failed. Please fix the issues and try again."
    exit 1
fi

echo "✅ Pre-commit checks passed successfully!"
exit 0
EOF
            ;;
        "pre-push")
            cat > "$hook_file" << 'EOF'
#!/bin/sh
# Pre-push hook for JFrog CLI Evidence
# This hook runs comprehensive checks (formatting, vetting, linting, and tests) before pushing

set -e

echo "Running pre-push checks..."

# Change to the repository root directory
cd "$(git rev-parse --show-toplevel)"

# Run the pre-push target from Makefile
# This runs: fmt, vet, lint, and test
make pre-push

if [ $? -ne 0 ]; then
    echo "❌ Pre-push checks failed. Please fix the issues before pushing."
    exit 1
fi

echo "✅ Pre-push checks passed successfully! Proceeding with push..."
exit 0
EOF
            ;;
        *)
            echo -e "${RED}Unknown hook: $hook_name${NC}"
            return 1
            ;;
    esac
    
    # Make the hook executable
    chmod +x "$hook_file"
    echo -e "${GREEN}✓ $hook_name hook installed${NC}"
}

# Install hooks
echo ""
echo "Installing Git hooks..."
echo "------------------------"

# Create pre-commit hook
create_hook "pre-commit"

# Create pre-push hook
create_hook "pre-push"

echo ""
echo -e "${GREEN}Git hooks setup complete!${NC}"
echo ""
echo "The following hooks have been installed:"
echo "  • pre-commit: Runs fmt, vet, and test-short"
echo "  • pre-push: Runs fmt, vet, lint, and test (full check)"
echo ""
echo "To skip hooks temporarily, use:"
echo "  • git commit --no-verify"
echo "  • git push --no-verify"
echo ""
echo "To uninstall hooks, simply delete the files:"
echo "  • rm $HOOKS_DIR/pre-commit"
echo "  • rm $HOOKS_DIR/pre-push"
