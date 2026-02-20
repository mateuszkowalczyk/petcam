#!/bin/bash

echo "Uninstalling petcam services..."

# 1. Stop and disable petcam service
sudo systemctl stop petcam.service 2>/dev/null
sudo systemctl disable petcam.service 2>/dev/null

# 2. Remove petcam service file
sudo rm -f /etc/systemd/system/petcam.service

# 3. Stop and disable LED permissions service
sudo systemctl stop led-perm.service 2>/dev/null
sudo systemctl disable led-perm.service 2>/dev/null

# 4. Remove LED permissions service file
sudo rm -f /etc/systemd/system/led-perm.service

# 5. Reload systemd
sudo systemctl daemon-reload

echo "Uninstallation complete!"
echo "Note: The dtparam=act_led_trigger=none line in /boot/firmware/config.txt was not removed."
echo "If you no longer need it, remove it manually."
