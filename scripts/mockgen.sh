#!/bin/bash
# Mock generation script for JFrog CLI Evidence

set -e

# Check if mockgen is installed
if ! command -v mockgen &> /dev/null; then
    echo "mockgen not found. Installing..."
    go install go.uber.org/mock/mockgen@latest
fi

# Get the source file from the argument
SOURCE_FILE="$1"

if [ -z "$SOURCE_FILE" ]; then
    echo "Usage: $0 <source_file.go>"
    exit 1
fi

# Resolve to absolute path if relative
if [[ ! "$SOURCE_FILE" = /* ]]; then
    SOURCE_FILE="$(pwd)/$SOURCE_FILE"
fi

# Get the directory and filename
DIR=$(dirname "$SOURCE_FILE")
FILENAME=$(basename "$SOURCE_FILE" .go)

# Determine the package name from the source file
PACKAGE=$(grep "^package " "$SOURCE_FILE" | head -1 | awk '{print $2}')

# Determine project root (look for go.mod)
if [ -n "$PROJECT_DIR" ]; then
    PROJECT_ROOT="$PROJECT_DIR"
else
    PROJECT_ROOT=$(pwd)
    while [ "$PROJECT_ROOT" != "/" ]; do
        if [ -f "$PROJECT_ROOT/go.mod" ]; then
            break
        fi
        PROJECT_ROOT=$(dirname "$PROJECT_ROOT")
    done
fi

# Create mocks directory if it doesn't exist
MOCK_DIR="$PROJECT_ROOT/mocks"
mkdir -p "$MOCK_DIR"

# Generate the mock file name
# Use relative path from project root for better organization
REL_PATH=${SOURCE_FILE#$PROJECT_ROOT/}
REL_DIR=$(dirname "$REL_PATH")
MOCK_FILE="$MOCK_DIR/mock_${FILENAME}.go"

echo "Generating mock for $SOURCE_FILE..."
echo "Output: $MOCK_FILE"

# Generate the mock
mockgen -source="$SOURCE_FILE" \
        -destination="$MOCK_FILE" \
        -package=mocks \
        -aux_files=""

echo "Mock generation complete for $FILENAME"
