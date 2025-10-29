#!/bin/sh
set -e

# Stop the service if it's running
if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet bem; then
        echo "Stopping BEM server..."
        systemctl stop bem || true
    fi

    if systemctl is-enabled --quiet bem 2>/dev/null; then
        echo "Disabling BEM server..."
        systemctl disable bem || true
    fi
fi

echo "BEM server stopped and disabled."
