#!/bin/bash

# Voice Ferry Web UI Development Helper Script
# Provides quick development commands and utilities

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WEB_UI_URL="http://localhost:3000"
DEV_PORT="3000"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[DEV]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if port is available
check_port() {
    local port="$1"
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 1  # Port is in use
    else
        return 0  # Port is available
    fi
}

# Function to setup development environment
setup_dev() {
    print_status "Setting up development environment..."
    
    # Check if .env exists
    if [ ! -f ".env" ]; then
        print_status "Creating .env file from template..."
        cp .env.example .env
        print_warning "Please review and edit .env file with your settings"
    fi
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        print_status "Installing Node.js dependencies..."
        npm install
    fi
    
    # Create necessary directories
    mkdir -p logs data config backups
    
    print_success "Development environment setup complete"
}

# Function to start development server
start_dev() {
    print_status "Starting development server..."
    
    if ! check_port $DEV_PORT; then
        print_error "Port $DEV_PORT is already in use"
        print_status "Processes using port $DEV_PORT:"
        lsof -Pi :$DEV_PORT -sTCP:LISTEN
        exit 1
    fi
    
    # Set development environment
    export NODE_ENV=development
    export LOG_LEVEL=debug
    
    print_status "Starting server on port $DEV_PORT..."
    node server.js
}

# Function to start with file watching
start_watch() {
    print_status "Starting development server with file watching..."
    
    if command -v nodemon >/dev/null 2>&1; then
        nodemon server.js
    else
        print_warning "nodemon not found. Installing globally..."
        npm install -g nodemon
        nodemon server.js
    fi
}

# Function to run linting
run_lint() {
    print_status "Running code linting..."
    
    if command -v eslint >/dev/null 2>&1; then
        eslint . --ext .js --fix
    else
        print_warning "ESLint not found. Installing..."
        npm install --save-dev eslint
        npx eslint . --ext .js --fix
    fi
}

# Function to run tests
run_tests() {
    print_status "Running tests..."
    
    # Unit tests
    if [ -f "package.json" ] && grep -q '"test"' package.json; then
        npm test
    else
        print_warning "No test script found in package.json"
    fi
    
    # Integration tests
    if [ -f "test-deployment.sh" ]; then
        print_status "Running deployment tests..."
        ./test-deployment.sh
    fi
}

# Function to generate sample data
generate_sample_data() {
    print_status "Generating sample data for development..."
    
    # Create sample configuration
    cat > config/sample.yml << 'EOF'
# Sample B2BUA Configuration for Development
sip:
  listen_address: "127.0.0.1"
  listen_port: 5060
  transport: "udp"

b2bua:
  max_concurrent_calls: 10
  call_timeout: 60

redis:
  host: "localhost"
  port: 6379

routing:
  default_route:
    enabled: true
    target: "sip:test@127.0.0.1:5080"

logging:
  level: "debug"
EOF

    # Create sample users
    cat > data/users.json << 'EOF'
[
  {
    "id": 1,
    "username": "admin",
    "email": "admin@voice-ferry.local",
    "role": "admin",
    "created_at": "2025-05-29T00:00:00Z"
  },
  {
    "id": 2,
    "username": "operator",
    "email": "operator@voice-ferry.local", 
    "role": "operator",
    "created_at": "2025-05-29T00:00:00Z"
  }
]
EOF

    print_success "Sample data generated"
}

# Function to clean development environment
clean_dev() {
    print_status "Cleaning development environment..."
    
    # Stop any running processes
    pkill -f "node server.js" || true
    
    # Clean logs
    rm -rf logs/*
    rm -rf data/temp/*
    
    # Clean npm cache
    npm cache clean --force
    
    print_success "Development environment cleaned"
}

# Function to show development status
show_status() {
    print_status "Development Environment Status"
    echo "========================================"
    
    # Check Node.js
    if command -v node >/dev/null 2>&1; then
        echo "Node.js: $(node --version)"
    else
        echo "Node.js: Not installed"
    fi
    
    # Check npm
    if command -v npm >/dev/null 2>&1; then
        echo "npm: $(npm --version)"
    else
        echo "npm: Not installed"
    fi
    
    # Check dependencies
    if [ -d "node_modules" ]; then
        echo "Dependencies: Installed"
    else
        echo "Dependencies: Not installed"
    fi
    
    # Check environment file
    if [ -f ".env" ]; then
        echo "Environment: Configured"
    else
        echo "Environment: Not configured"
    fi
    
    # Check port availability
    if check_port $DEV_PORT; then
        echo "Port $DEV_PORT: Available"
    else
        echo "Port $DEV_PORT: In use"
    fi
    
    echo "========================================"
}

# Function to open development tools
open_tools() {
    print_status "Opening development tools..."
    
    # Open browser
    if command -v open >/dev/null 2>&1; then
        open "$WEB_UI_URL"
    elif command -v xdg-open >/dev/null 2>&1; then
        xdg-open "$WEB_UI_URL"
    else
        print_status "Please open $WEB_UI_URL in your browser"
    fi
    
    # Open VS Code if available
    if command -v code >/dev/null 2>&1; then
        code .
    fi
}

# Function to show usage
show_usage() {
    echo "Voice Ferry Web UI Development Helper"
    echo ""
    echo "Usage: $0 {setup|start|watch|lint|test|sample|clean|status|tools}"
    echo ""
    echo "Commands:"
    echo "  setup   - Setup development environment"
    echo "  start   - Start development server"
    echo "  watch   - Start with file watching (nodemon)"
    echo "  lint    - Run code linting"
    echo "  test    - Run tests"
    echo "  sample  - Generate sample data"
    echo "  clean   - Clean development environment"
    echo "  status  - Show development status"
    echo "  tools   - Open development tools"
}

# Main execution
case "$1" in
    setup)
        setup_dev
        ;;
    start)
        setup_dev
        start_dev
        ;;
    watch)
        setup_dev
        start_watch
        ;;
    lint)
        run_lint
        ;;
    test)
        run_tests
        ;;
    sample)
        generate_sample_data
        ;;
    clean)
        clean_dev
        ;;
    status)
        show_status
        ;;
    tools)
        open_tools
        ;;
    *)
        show_usage
        exit 1
        ;;
esac
