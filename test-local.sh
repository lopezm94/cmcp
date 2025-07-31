#!/bin/bash
# Local test runner - runs tests without containers
# Useful for quick testing during development

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== CMCP Local Test Runner ===${NC}"
echo "Running tests locally (without containers)"
echo ""

# Track overall test status
OVERALL_STATUS=0

# Function to run a test section
run_test_section() {
    local section_name="$1"
    local command="$2"
    
    echo -e "${BLUE}=== $section_name ===${NC}"
    if eval "$command"; then
        echo -e "${GREEN}✓ $section_name passed${NC}"
    else
        echo -e "${RED}✗ $section_name failed${NC}"
        OVERALL_STATUS=1
    fi
    echo ""
}

# Build the binary
echo "Building cmcp..."
go build -o ./cmcp

# 1. Run unit tests
run_test_section "Unit Tests" "go test ./..."

# 2. The unit tests above already cover dry-run functionality

# Summary
echo -e "${BLUE}=== Test Summary ===${NC}"
if [ $OVERALL_STATUS -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
else
    echo -e "${RED}❌ Some tests failed. Check output above for details.${NC}"
fi

exit $OVERALL_STATUS