#!/bin/bash

cd /drone/src/

DRONE_TAG=$1

if [[ -z $DRONE_TAG ]]
then
  echo "Missing DRONE_TAG"
  exit 1
fi

echo "Find and replace DRONE_TAG..."
sed -i "s/DRONE_TAG/${DRONE_TAG}" ./Chart/Chart.yaml
sed -i "s/DRONE_TAG/${DRONE_TAG}" ./Chart/values.yaml
sed -i "s/DRONE_BUILD_NUMBER/${DRONE_BUILD_NUMBER}" ./Chart/Chart.yaml
sed -i "s/DRONE_BUILD_NUMBER/${DRONE_BUILD_NUMBER}" ./Chart/values.yaml

echo "Packaging helm chart..."
helm package ./Chart/ --version $DRONE_TAG --app-version $DRONE_BUILD_NUMBER

echo "Pulling down chart repo..."
mkdir -p helm-repo
cd helm-repo
git clone git@github.com:SupportTools/helm-chart.git .

echo "Copying package into repo..."
cp /drone/src/kubebackup-*.tgz .

echo "Reindexing repo..."
helm repo index --url https://charts.support.tools/ --merge index.yaml .

echo "Publishing to Chart repo..."
git add .
git commit -m "Publishing KubeBackup ${DRONE_TAG}"
git push