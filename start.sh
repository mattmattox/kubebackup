#!/bin/bash
set -e
echo "Setting up cron..."
echo "$CRON_SCHEDULE /sync.sh" >> /var/spool/cron/crontabs/root
crond -l 2 -f
