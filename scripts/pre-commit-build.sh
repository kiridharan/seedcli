#!/bin/bash
set -e
echo "Running Go build check..."
go build ./...
echo "Build successful!"
