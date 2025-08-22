#!/bin/bash

# Image management script for CMCP test base image
# Commands: build, ensure, status, clean

set -e

# Configuration
IMAGE_NAME="cmcp-test-base"
DOCKERFILE_PATH="tests/Dockerfile.base"

# Detect container runtime
if command -v podman >/dev/null 2>&1; then
    RUNTIME="podman"
    # Podman uses localhost prefix for local images
    FULL_IMAGE_NAME="localhost/$IMAGE_NAME"
elif command -v docker >/dev/null 2>&1; then
    RUNTIME="docker"
    FULL_IMAGE_NAME="$IMAGE_NAME"
else
    echo "Error: Neither Podman nor Docker found"
    exit 1
fi

# Function to check if image exists
image_exists() {
    if $RUNTIME image inspect "$FULL_IMAGE_NAME" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to build the image
build_image() {
    echo "Building base test image with $RUNTIME..."
    echo "Image name: $FULL_IMAGE_NAME"
    
    if ! [ -f "$DOCKERFILE_PATH" ]; then
        echo "Error: Dockerfile not found at $DOCKERFILE_PATH"
        exit 1
    fi
    
    $RUNTIME build -f "$DOCKERFILE_PATH" -t "$FULL_IMAGE_NAME" .
    
    if [ $? -eq 0 ]; then
        echo "✓ Base image built successfully: $FULL_IMAGE_NAME"
        return 0
    else
        echo "✗ Failed to build base image"
        return 1
    fi
}

# Main command handler
case "${1:-help}" in
    build)
        build_image
        ;;
        
    ensure)
        if image_exists; then
            echo "✓ Base image already exists: $FULL_IMAGE_NAME"
            exit 0
        else
            echo "Base image not found, building..."
            build_image
        fi
        ;;
        
    status)
        if image_exists; then
            echo "✓ Base image exists: $FULL_IMAGE_NAME"
            echo ""
            echo "Image details:"
            $RUNTIME image inspect "$FULL_IMAGE_NAME" | jq -r '.[0] | {
                Id: .Id[0:12],
                Created: .Created,
                Size: .Size,
                RepoTags: .RepoTags
            }'
            exit 0
        else
            echo "✗ Base image does not exist: $FULL_IMAGE_NAME"
            echo "Run './tests/manage-image.sh build' to create it"
            exit 1
        fi
        ;;
        
    clean)
        if image_exists; then
            echo "Removing base image: $FULL_IMAGE_NAME"
            $RUNTIME rmi "$FULL_IMAGE_NAME"
            if [ $? -eq 0 ]; then
                echo "✓ Base image removed successfully"
            else
                echo "✗ Failed to remove base image"
                exit 1
            fi
        else
            echo "Base image does not exist, nothing to clean"
        fi
        ;;
        
    help|*)
        echo "CMCP Test Base Image Manager"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  build   - Build the base image (rebuilds even if exists)"
        echo "  ensure  - Build the base image only if it doesn't exist"
        echo "  status  - Check if the base image exists and show details"
        echo "  clean   - Remove the base image"
        echo ""
        echo "Runtime: $RUNTIME"
        echo "Image: $FULL_IMAGE_NAME"
        ;;
esac