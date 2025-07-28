#!/bin/bash

# Dependencies installer script
# This script installs build dependencies on Linux systems (Fedora/RHEL/Debian/Ubuntu)
#
# Usage:
#   ./hack/dependencies.sh                          # Install dependencies automatically
#
# Supports:
#   - Fedora/RHEL (dnf): Uses rpm/podman.spec with builddep
#   - CentOS/RHEL (yum): Uses rpm/podman.spec with yum-builddep
#   - Debian/Ubuntu (apt-get): Uses build/debian-packages.txt

set -euo pipefail

echo "Checking build dependencies..."

if [ "$(uname -s)" = "Linux" ]; then
    if command -v dnf >/dev/null 2>&1; then
        echo "Fedora/RHEL detected - installing build dependencies..."
        if [ -f rpm/podman.spec ]; then
            sudo dnf -y builddep rpm/podman.spec || echo "Warning: Some dependencies may have failed to install"
        else
            echo "Warning: rpm/podman.spec not found"
        fi
    elif command -v yum >/dev/null 2>&1; then
        echo "CentOS/RHEL (yum) detected - installing build dependencies..."
        if [ -f rpm/podman.spec ]; then
            sudo yum-builddep -y rpm/podman.spec || echo "Warning: Some dependencies may have failed to install"
        else
            echo "Warning: rpm/podman.spec not found"
        fi
    elif command -v apt-get >/dev/null 2>&1; then
        echo "Debian/Ubuntu detected - installing build dependencies..."
        if [ -f build/debian-packages.txt ]; then
            packages=$(cat build/debian-packages.txt | tr '\n' ' ')
            sudo apt-get update && sudo apt-get install -y $packages \
                || echo "Warning: Some dependencies may have failed to install"
        else
            echo "Warning: build/debian-packages.txt not found"
        fi
    else
        echo "Unsupported Linux distribution - skipping dependency installation"
    fi
else
    echo "Non-Linux system detected - skipping dependency installation"
fi

echo "Dependency installation completed."