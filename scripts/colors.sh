#!/bin/bash

# Color definitions for consistent output across scripts
# Usage: source this file and use the color functions

# Reset
RESET='\033[0m'

# Regular Colors
BLACK='\033[0;30m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[0;37m'

# Bold Colors
BOLD_BLACK='\033[1;30m'
BOLD_RED='\033[1;31m'
BOLD_GREEN='\033[1;32m'
BOLD_YELLOW='\033[1;33m'
BOLD_BLUE='\033[1;34m'
BOLD_PURPLE='\033[1;35m'
BOLD_CYAN='\033[1;36m'
BOLD_WHITE='\033[1;37m'

# Helper functions
print_header() {
    echo -e "${BOLD_CYAN}$1${RESET}"
}

print_success() {
    echo -e "${BOLD_GREEN}✅ $1${RESET}"
}

print_error() {
    echo -e "${BOLD_RED}✗ $1${RESET}"
}

print_warning() {
    echo -e "${BOLD_YELLOW}⚠️  $1${RESET}"
}

print_info() {
    echo -e "${CYAN}ℹ️  $1${RESET}"
}

print_step() {
    echo -e "${BLUE}→ $1${RESET}"
}

print_detail() {
    echo -e "   ${WHITE}$1${RESET}"
}

print_command() {
    echo -e "   ${BOLD_WHITE}\$ $1${RESET}"
}