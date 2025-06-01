#!/bin/bash
# CI/CD Health Check Script

set -e

echo "🔍 CI/CD Health Check for Voice Ferry"
echo "======================================"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ Not in a git repository"
    exit 1
fi

echo "✅ Git repository detected"

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    exit 1
fi

echo "✅ Go $(go version | awk '{print $3}') detected"

# Check go.mod
if [ ! -f "go.mod" ]; then
    echo "❌ go.mod not found"
    exit 1
fi

echo "✅ go.mod found"

# Check module name
MODULE_NAME=$(grep "^module " go.mod | awk '{print $2}')
echo "📦 Module: $MODULE_NAME"

# Check if CI/CD workflow exists
if [ ! -f ".github/workflows/ci-cd.yml" ]; then
    echo "❌ CI/CD workflow not found"
    exit 1
fi

echo "✅ CI/CD workflow found"

# Check golangci-lint config
if [ ! -f ".golangci.yml" ]; then
    echo "❌ golangci-lint config not found"
    exit 1
fi

echo "✅ golangci-lint config found"

# Validate YAML files
echo "🔍 Validating YAML files..."

if command -v yamllint &> /dev/null; then
    yamllint .github/workflows/ci-cd.yml
    yamllint .golangci.yml
    echo "✅ YAML validation passed"
else
    echo "⚠️  yamllint not installed, skipping YAML validation"
fi

# Check for test files
TEST_FILES=$(find . -name "*_test.go" -not -path "./vendor/*" | wc -l)
echo "📊 Found $TEST_FILES test files"

# Check for common CI/CD issues
echo "🔍 Checking for common CI/CD issues..."

# Check if the main application exists
if [ ! -d "cmd/b2bua" ]; then
    echo "❌ Main application directory cmd/b2bua not found"
    exit 1
fi

echo "✅ Main application directory found"

# Check if Dockerfile exists
if [ ! -f "Dockerfile" ]; then
    echo "❌ Dockerfile not found"
    exit 1
fi

echo "✅ Dockerfile found"

# Check for security issues in workflow
if grep -q "secrets\." .github/workflows/ci-cd.yml; then
    echo "⚠️  Secrets usage detected in workflow (review for security)"
fi

# Run basic Go checks
echo "🔍 Running basic Go checks..."

# Check if modules can be downloaded
if ! go mod download; then
    echo "❌ Failed to download Go modules"
    exit 1
fi

echo "✅ Go modules downloaded successfully"

# Check if code can be formatted
if ! go fmt ./...; then
    echo "❌ Go formatting failed"
    exit 1
fi

echo "✅ Go formatting passed"

# Check if code compiles
if ! go build ./...; then
    echo "❌ Go build failed"
    exit 1
fi

echo "✅ Go build successful"

# Run tests
echo "🔍 Running tests..."
if ! go test -short ./...; then
    echo "❌ Tests failed"
    exit 1
fi

echo "✅ Tests passed"

# Check for vulnerabilities if govulncheck is available
if command -v govulncheck &> /dev/null; then
    echo "🔍 Running vulnerability check..."
    if ! govulncheck ./...; then
        echo "⚠️  Vulnerability check found issues"
    else
        echo "✅ No vulnerabilities found"
    fi
else
    echo "⚠️  govulncheck not installed, skipping vulnerability check"
fi

echo ""
echo "🎉 CI/CD Health Check Complete!"
echo "================================="
echo "✅ All checks passed successfully"
echo ""
echo "📋 Summary:"
echo "  - Module: $MODULE_NAME"
echo "  - Test files: $TEST_FILES"
echo "  - Go version: $(go version | awk '{print $3}')"
echo ""
echo "🚀 Your CI/CD pipeline is ready!"
