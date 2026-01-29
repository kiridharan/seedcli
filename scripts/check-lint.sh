#!/bin/bash

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null
then
    echo "golangci-lint could not be found."
    echo "Installing golangci-lint..."
    # Install to ./bin to avoid needing sudo
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v1.55.2
    LINTER="./bin/golangci-lint"
else
    LINTER="golangci-lint"
fi

echo "Running linter..."
$LINTER run ./...
