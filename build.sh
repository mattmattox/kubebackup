#!/bin/bash

help() {

  echo "Build Script
  Usage: build.sh -t DRONE_BRANCH -b DRONE_BUILD_NUMBER -e ENV
  All flags are optional
  -t    Drone Tag (v1.2.3)
  -b    Drone build number (10)
  -e    Environment (dev|production)"

}

while getopts ":d:s:r:fph" opt; do
  case $opt in
    t)
      DRONE_BRANCH="${OPTARG}"
      ;;
    b)
      DRONE_BUILD_NUMBER="${OPTARG}"
      ;;
    b)
      ENV="${OPTARG}"
      ;;
    h)
      help && exit 0
      ;;
    :)
      techo "Option -$OPTARG requires an argument."
      exit 1
      ;;
    *)
      help && exit 0
  esac
done

cd /drone/src/

if [[ -z $DRONE_BRANCH ]] || [[ -z $DRONE_BUILD_NUMBER ]] || [[ -z $ENV]]
then
  help
  exit 1
fi

echo "Find and replace DRONE_BRANCH..."
sed -i "s/DRONE_BRANCH/${DRONE_BRANCH}" ./Chart/Chart.yaml
sed -i "s/DRONE_BRANCH/${DRONE_BRANCH}" ./Chart/values.yaml
sed -i "s/DRONE_BUILD_NUMBER/${DRONE_BUILD_NUMBER}" ./Chart/Chart.yaml
sed -i "s/DRONE_BUILD_NUMBER/${DRONE_BUILD_NUMBER}" ./Chart/values.yaml

echo "Packaging helm chart..."
helm package ./Chart/ --version $DRONE_BRANCH --app-version $DRONE_BUILD_NUMBER

echo "Pulling down chart repo..."
mkdir -p helm-repo
cd helm-repo
if [[ ${ENV} == "production" ]]
then
  git clone git@github.com:SupportTools/helm-chart.git .
elif [[ ${ENV} == "dev" ]]
then
  git clone git@github.com:SupportTools/helm-chart-dev.git .
else
  echo "Unknown Environment"
fi

echo "Copying package into repo..."
cp /drone/src/kubebackup-*.tgz .

echo "Reindexing repo..."
if [[ ${ENV} == "production" ]]
then
  helm repo index --url https://charts.support.tools/ --merge index.yaml .
elif [[ ${ENV} == "dev" ]]
then
  helm repo index --url https://charts-dev.support.tools/ --merge index.yaml .
else
  echo "Unknown Environment"
fi


echo "Publishing to Chart repo..."
git add .
git commit -m "Publishing KubeBackup ${DRONE_BRANCH}"
git push