#!/bin/bash

# Pre-commit hook for CodeAtlas
# This hook runs tests and checks coverage before allowing commits
# 
# To install: ln -s ./scripts/pre-commit-hook.sh .git/hooks/pre-commit

set -e

echo "🔍 Running pre-commit checks..."

# Check if Go files were modified
GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$GO_FILES" ]; then
    echo "✅ No Go files modified, skipping tests"
    exit 0
fi

echo "📝 Go files modified:"
echo "$GO_FILES"
echo ""

# Run go fmt
echo "🎨 Running go fmt..."
gofmt -w $GO_FILES
git add $GO_FILES

# Run go vet
echo "🔎 Running go vet..."
go vet ./...

# Run tests
echo "🧪 Running tests..."
go test ./... -short

# Optional: Check coverage (uncomment to enable)
# echo "📊 Checking coverage..."
# go test ./... -coverprofile=coverage.out -covermode=atomic
# COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
# THRESHOLD=50
# if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
#     echo "❌ Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%"
#     rm coverage.out
#     exit 1
# fi
# rm coverage.out

echo ""
echo "✅ All pre-commit checks passed!"
exit 0
