#!/bin/sh
set -e

# Create bem user and group if they don't exist
if ! getent group bem >/dev/null; then
    groupadd -r bem
fi

if ! getent passwd bem >/dev/null; then
    useradd -r -g bem -d /var/lib/bem -s /sbin/nologin -c "BEM Server" bem
fi

# Set ownership of data directories
chown -R bem:bem /var/lib/bem
chmod 755 /var/lib/bem

# Reload systemd if available
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

echo "BigFix Enterprise Mobile (BEM) Server installed successfully!"
echo ""
echo "Next steps:"
echo "1. Create a configuration file at /etc/bem/bem.json"
echo "2. Generate TLS certificates (required for operation)"
echo "3. Start the service: systemctl start bem"
echo "4. Enable automatic startup: systemctl enable bem"
echo ""
echo "For more information, see: /usr/share/doc/bigfix-mobile-enterprise/README.md"
