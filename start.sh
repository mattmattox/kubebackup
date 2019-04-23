#!/bin/ash

set -e

export AWS_ACCESS_KEY_ID=$KEY
export AWS_SECRET_ACCESS_KEY=$SECRET
export AWS_DEFAULT_REGION=$REGION
export CLUSTER=$CLUSTER

echo "Creating AWS creds..."
mkdir -p ~/.aws/
echo '[default]' > ~/.aws/credentials
echo 'aws_access_key_id='"$KEY" >> ~/.aws/credentials
echo 'aws_secret_access_key='"$SECRET" >> ~/.aws/credentials
echo '[default]' > ~/.aws/config
echo 'region='"$REGION" >> ~/.aws/config
echo 'output=json' >> ~/.aws/config

echo "Creating kubeconfig..."
mkdir -p ~/.kube/
kubectl get secret -n kube-system kube-admin -o jsonpath={.data.Config} | base64 -d > ~/.kube/config
sed -i -e 's/[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}/127.0.0.1/g' ~/.kube/config

#echo "Running once during startup..."
#/sync.sh

echo "Setting up cron..."
echo "$CRON_SCHEDULE /sync.sh" >> /var/spool/cron/crontabs/root
crond -l 2 -f
