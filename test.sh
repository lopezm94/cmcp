#!/bin/bash

# CMCP Test Suite
# Usage: 
#   ./test.sh                    # Run all tests (default)
#   ./test.sh logging            # Run only logging test
#   ./test.sh logging install    # Run logging and install tests
#   ./test.sh unit comprehensive # Run unit and comprehensive tests

# Available tests: unit, comprehensive, install, logging, web, online, config, non-interactive

echo "=== CMCP Test Runner ==="

# Parse arguments to determine which tests to run
RUN_UNIT=0
RUN_COMPREHENSIVE=0
RUN_INSTALL=0
RUN_LOGGING=0
RUN_WEB=0
RUN_ONLINE=0
RUN_CONFIG=0
RUN_NON_INTERACTIVE=0

# Default to all tests if no arguments
if [ $# -eq 0 ]; then
    RUN_UNIT=1
    RUN_COMPREHENSIVE=1
    RUN_INSTALL=1
    RUN_LOGGING=1
    RUN_WEB=1
    RUN_ONLINE=1
    RUN_CONFIG=1
    RUN_NON_INTERACTIVE=1
    echo "Running all tests (use './test.sh <test-names>' to run specific tests)"
    echo "Available tests: unit comprehensive install logging web online config non-interactive"
else
    # Parse requested tests
    for arg in "$@"; do
        case $arg in
            unit)
                RUN_UNIT=1
                ;;
            comprehensive)
                RUN_COMPREHENSIVE=1
                ;;
            install)
                RUN_INSTALL=1
                ;;
            logging)
                RUN_LOGGING=1
                ;;
            web)
                RUN_WEB=1
                ;;
            config)
                RUN_CONFIG=1
                ;;
            non-interactive|noninteractive|ni)
                RUN_NON_INTERACTIVE=1
                ;;
            online)
                RUN_ONLINE=1
                ;;
            *)
                echo "Unknown test: $arg"
                echo "Available tests: unit comprehensive install logging web online config non-interactive"
                exit 1
                ;;
        esac
    done
    echo "Running selected tests: $@"
fi

echo "Automatically detecting container runtime..."

# Check for Podman first, then Docker
if command -v podman >/dev/null 2>&1; then
    echo "Using Podman"
    echo "=== Running CMCP Tests in Podman ==="
    echo "This will test all functionality in an isolated container."
    echo ""

    # Build the test image
    echo "Building test image..."
    podman build -f tests/Dockerfile.test -t cmcp-test .

    if [ $? -ne 0 ]; then
        echo "❌ Failed to build test image"
        exit 1
    fi

    echo "Running tests..."

    # Build dynamic test command based on requested tests
    TEST_CMD=""
    
    # Always build cmcp first
    TEST_CMD="export PATH=/usr/local/go/bin:\$PATH && cd /app && go build -o /tmp/cmcp && export PATH=/tmp:\$PATH && cd /tmp && cp -r /app/tests ."
    
    # Add requested tests
    if [ $RUN_UNIT -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Unit Tests ===' && cd /app && go test ./... -v"
    fi
    
    if [ $RUN_COMPREHENSIVE -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Comprehensive Tests ===' && sed 's|\./cmcp|/tmp/cmcp|g; s|set -e|set +e|g' /app/tests/test-comprehensive.sh > /tmp/test-comprehensive.sh && chmod +x /tmp/test-comprehensive.sh && /tmp/test-comprehensive.sh"
    fi
    
    if [ $RUN_INSTALL -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Install/Uninstall Tests ===' && chmod +x /tmp/tests/test-install-scripts.sh && /tmp/tests/test-install-scripts.sh || true"
    fi
    
    if [ $RUN_LOGGING -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Automatic Logging Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Rebuilding cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && ls -la /tmp/cmcp && cp /app/tests/test-logging.sh /tmp/test-logging.sh && chmod +x /tmp/test-logging.sh && cd /tmp && CMCP_BIN=/tmp/cmcp /tmp/test-logging.sh || true"
    fi
    
    if [ $RUN_WEB -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Web Install/Uninstall Tests ===' && cp /app/tests/test-web-install.sh /tmp/ && chmod +x /tmp/test-web-install.sh && /tmp/test-web-install.sh || true"
    fi
    
    if [ $RUN_ONLINE -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Online Command Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Rebuilding cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && cp /app/tests/test-online.sh /tmp/test-online.sh && chmod +x /tmp/test-online.sh && cd /tmp && CMCP_BIN=/tmp/cmcp /tmp/test-online.sh || true"
    fi

    if [ $RUN_CONFIG -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Config Preservation Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Building cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && cp /app/tests/test-config.sh /tmp/test-config.sh && chmod +x /tmp/test-config.sh && cd /tmp && CMCP_BIN=/tmp/cmcp ./test-config.sh || true"
    fi

    if [ $RUN_NON_INTERACTIVE -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Non-Interactive Mode Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Building cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && cp /app/tests/test-non-interactive.sh /tmp/test-non-interactive.sh && chmod +x /tmp/test-non-interactive.sh && cd /tmp && ./test-non-interactive.sh || true"
    fi

    # Run tests in container with proper binary location
    podman run --rm \
        -v ./:/app:ro \
        --tmpfs /tmp \
        --tmpfs /root \
        -e HOME=/root \
        cmcp-test \
        /bin/bash -c "$TEST_CMD"
elif command -v docker >/dev/null 2>&1; then
    echo "Using Docker"
    echo "=== Running CMCP Tests in Docker ==="
    echo "This will test all functionality in an isolated container."
    echo ""

    # Build the test image
    echo "Building test image..."
    docker build -f tests/Dockerfile.test -t cmcp-test .

    if [ $? -ne 0 ]; then
        echo "❌ Failed to build test image"
        exit 1
    fi

    echo "Running tests..."

    # Build dynamic test command based on requested tests
    TEST_CMD=""
    
    # Always build cmcp first
    TEST_CMD="export PATH=/usr/local/go/bin:\$PATH && cd /app && go build -o /tmp/cmcp && export PATH=/tmp:\$PATH && cd /tmp && cp -r /app/tests ."
    
    # Add requested tests
    if [ $RUN_UNIT -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Unit Tests ===' && cd /app && go test ./... -v"
    fi
    
    if [ $RUN_COMPREHENSIVE -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Comprehensive Tests ===' && sed 's|\./cmcp|/tmp/cmcp|g; s|set -e|set +e|g' /app/tests/test-comprehensive.sh > /tmp/test-comprehensive.sh && chmod +x /tmp/test-comprehensive.sh && /tmp/test-comprehensive.sh"
    fi
    
    if [ $RUN_INSTALL -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Install/Uninstall Tests ===' && chmod +x /tmp/tests/test-install-scripts.sh && /tmp/tests/test-install-scripts.sh || true"
    fi
    
    if [ $RUN_LOGGING -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Automatic Logging Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Rebuilding cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && ls -la /tmp/cmcp && cp /app/tests/test-logging.sh /tmp/test-logging.sh && chmod +x /tmp/test-logging.sh && cd /tmp && CMCP_BIN=/tmp/cmcp /tmp/test-logging.sh || true"
    fi
    
    if [ $RUN_WEB -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Web Install/Uninstall Tests ===' && cp /app/tests/test-web-install.sh /tmp/ && chmod +x /tmp/test-web-install.sh && /tmp/test-web-install.sh || true"
    fi
    
    if [ $RUN_ONLINE -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Online Command Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Rebuilding cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && cp /app/tests/test-online.sh /tmp/test-online.sh && chmod +x /tmp/test-online.sh && cd /tmp && CMCP_BIN=/tmp/cmcp /tmp/test-online.sh || true"
    fi

    if [ $RUN_CONFIG -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Config Preservation Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Building cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && cp /app/tests/test-config.sh /tmp/test-config.sh && chmod +x /tmp/test-config.sh && cd /tmp && CMCP_BIN=/tmp/cmcp ./test-config.sh || true"
    fi

    if [ $RUN_NON_INTERACTIVE -eq 1 ]; then
        TEST_CMD="$TEST_CMD && echo '' && echo '=== Running Non-Interactive Mode Tests ===' && if [ ! -f /tmp/cmcp ]; then echo 'Building cmcp binary...' && cd /app && go build -o /tmp/cmcp; fi && cp /app/tests/test-non-interactive.sh /tmp/test-non-interactive.sh && chmod +x /tmp/test-non-interactive.sh && cd /tmp && ./test-non-interactive.sh || true"
    fi

    # Run tests in container with proper binary location
    docker run --rm \
        -v $(pwd):/app:ro \
        --tmpfs /tmp \
        --tmpfs /root \
        -e HOME=/root \
        cmcp-test \
        /bin/bash -c "$TEST_CMD"
else
    echo "❌ Error: Neither Podman nor Docker found"
    echo "Please install Podman or Docker to run tests"
    exit 1
fi

# Get exit code
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ All tests passed!"
else
    echo ""
    echo "❌ Tests failed. Check output above for details."
fi

exit $EXIT_CODE