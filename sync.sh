#!/bin/ash

set -e

echo "$(date) - Start"
echo "Starting export..."
/backup.sh
echo "$(date) End"
