#!/bin/bash

echo "Building the project..."
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker
go build -o bin/migrate ./cmd/migrate
echo "Build complete."
