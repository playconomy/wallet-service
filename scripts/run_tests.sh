#!/bin/bash

# Script to run all tests for wallet-service

set -e  # Exit on any error

echo "Running service unit tests..."
go test -v ./internal/service

echo "Running handler unit tests..."
go test -v ./internal/server/handler

echo "Running middleware unit tests..."
go test -v ./internal/server/middleware

echo "Running utils tests..."
go test -v ./internal/utils

echo "Running integration tests (use -short to skip)..."
go test -v ./internal/test/integration

echo "All tests passed!"
