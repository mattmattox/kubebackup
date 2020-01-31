#!/bin/bash

set -e

export AWS_ACCESS_KEY_ID=$s3AccessKey
export AWS_SECRET_ACCESS_KEY=$s3SecretKey
export AWS_DEFAULT_REGION=$S3_REGION
export CLUSTER=$CLUSTER

echo "Creating AWS creds..."
mkdir -p ~/.aws/
echo '[default]' > ~/.aws/credentials
echo 'aws_access_key_id='"$KEY" >> ~/.aws/credentials
echo 'aws_secret_access_key='"$SECRET" >> ~/.aws/credentials
echo '[default]' > ~/.aws/config
echo 'region='"$REGION" >> ~/.aws/config
echo 'output=json' >> ~/.aws/config

#echo "Running once during startup..."
/sync.sh

echo "Setting up cron..."
echo "$CRON_SCHEDULE /sync.sh" >> /var/spool/cron/crontabs/root
crond -l 2 -f
