#!/bin/bash

# Build script for PicsApp

echo "Building PicsApp..."

# Build React frontend
echo "Building React frontend..."
npm run build

if [ $? -ne 0 ]; then
    echo "Error: React build failed"
    exit 1
fi

# Build Go backend
echo "Building Go backend..."
go build -o picsapp main.go

if [ $? -ne 0 ]; then
    echo "Error: Go build failed"
    exit 1
fi

echo "Build complete!"
echo "Run the server with: ./picsapp"
echo "Or set PORT environment variable: PORT=3000 ./picsapp"

