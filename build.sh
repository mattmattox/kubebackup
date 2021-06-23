#!/bin/bash

help() {

  echo "Build Script
  Usage: build.sh -b DRONE_BUILD_NUMBER -e Environment
  All flags are optional
  -b    Drone build number (10)
  -e    Environment (dev|production)"

}

while getopts ":b:r:e:h" opt; do
  case $opt in
    b)
      DRONE_BUILD_NUMBER="${OPTARG}"
      ;;
    r)
      RELEASE="${OPTARG}"
      ;;
    e)
      Environment="${OPTARG}"
      ;;
    h)
      help && exit 0
      ;;
    :)
      echo "Option -$OPTARG requires an argument."
      exit 1
      ;;
    *)
      help && exit 0
  esac
done

if [[ -z $Release ]]
then
  echo "Release must be set"
  exit 0
fi

echo "::Info::"
echo "Environment: $Environment"
echo "Release: $RELEASE"
echo "Build Number: $DRONE_BUILD_NUMBER"

cd /drone/src/

echo "Find and replace values..."
sed -i "s|RELEASE|${RELEASE}|g" ./Chart/Chart.yaml
sed -i "s|RELEASE|${RELEASE}|g" ./Chart/values.yaml
sed -i "s|DRONE_BUILD_NUMBER|${DRONE_BUILD_NUMBER}|g" ./Chart/Chart.yaml
sed -i "s|DRONE_BUILD_NUMBER|${DRONE_BUILD_NUMBER}|g" ./Chart/values.yaml

echo "::Chart::"
cat ./Chart/Chart.yaml
echo "::Values::"
cat ./Chart/values.yaml

echo "Packaging helm chart..."
helm package ./Chart/ --version $RELEASE --app-version $DRONE_BUILD_NUMBER

echo "Pulling down chart repo..."
mkdir -p helm-repo
cd helm-repo
if [[ ${Environment} == "production" ]]
then
  git clone git@github.com:SupportTools/helm-chart.git .
elif [[ ${Environment} == "dev" ]]
then
  git clone git@github.com:SupportTools/helm-chart-dev.git .
else
  echo "Unknown Environment"
fi

echo "Copying package into repo..."
cp /drone/src/kubebackup-*.tgz .

echo "Reindexing repo..."
if [[ ${Environment} == "production" ]]
then
  helm repo index --url https://charts.support.tools/ --merge index.yaml .
elif [[ ${Environment} == "dev" ]]
then
  helm repo index --url https://charts-dev.support.tools/ --merge index.yaml .
else
  echo "Unknown Environment"
fi


echo "Publishing to Chart repo..."
git add .
git commit -m "Publishing KubeBackup ${RELEASE}"
git push