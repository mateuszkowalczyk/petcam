#!/bin/bash
set -eo pipefail

# 1. Persist LED control at boot
echo 'dtparam=act_led_trigger=none' | sudo tee -a /boot/firmware/config.txt

# 2. Create a systemd service for LED permissions
sudo tee /etc/systemd/system/led-perm.service >/dev/null <<'EOF'
[Unit]
Description=Set ACT LED permissions

[Service]
Type=oneshot
ExecStart=/bin/chmod 666 /sys/class/leds/ACT/brightness

[Install]
WantedBy=multi-user.target
EOF

# 3. Enable the service
sudo systemctl enable led-perm.service

# 4. Reboot
sudo reboot
