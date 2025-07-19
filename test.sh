#!/bin/bash

echo "=== CMCP Test Runner ==="
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

    # Run tests in container with proper binary location
    podman run --rm \
        -v ./:/app:ro \
        --tmpfs /tmp \
        --tmpfs /root \
        -e HOME=/root \
        cmcp-test \
        /bin/bash -c "
            export PATH=/usr/local/go/bin:\$PATH &&
            cd /app && 
            go build -o /tmp/cmcp && 
            export PATH=/tmp:\$PATH &&
            cd /tmp &&
            cp -r /app/tests . &&
            # Replace ./cmcp with /tmp/cmcp and disable set -e in test script
            sed 's|\\\\./cmcp|/tmp/cmcp|g; s|set -e|set +e|g' /app/tests/test-comprehensive.sh > /tmp/test-comprehensive.sh &&
            chmod +x /tmp/tests/mock-mcp-server.sh &&
            chmod +x /tmp/test-comprehensive.sh &&
            chmod +x /tmp/tests/test-install-scripts.sh &&
            echo '=== Running Comprehensive Tests ===' &&
            /tmp/test-comprehensive.sh &&
            echo '' &&
            echo '=== Running Install/Uninstall Tests ===' &&
            /tmp/tests/test-install-scripts.sh
        "
elif command -v docker >/dev/null 2>&1; then
    echo "Using Docker"
    echo "=== Running CMCP Tests in Docker ==="
    echo "This will test all functionality in an isolated container."
    echo ""

    # Build and run tests
    docker-compose -f tests/docker-compose.test.yml up --build --abort-on-container-exit --force-recreate

    # Get exit code
    EXIT_CODE=$?

    # Clean up
    docker-compose -f tests/docker-compose.test.yml down
    exit $EXIT_CODE
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