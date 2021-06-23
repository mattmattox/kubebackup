#!/bin/bash

help() {

  echo "Build Script
  Usage: build.sh -b DRONE_BUILD_NUMBER -r Release -e Environment
  All flags are optional
  -b    Drone build number (10)
  -r    Release number (v0.1.2-rc1)
  -e    Environment (dev|production)"

}

while getopts ":b:r:e:h" opt; do
  case $opt in
    b)
      DRONE_BUILD_NUMBER="${OPTARG}"
      ;;
    r)
      Release="${OPTARG}"
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

cd /drone/src/

if [[ -z $RELEASE ]] || [[ -z $DRONE_BUILD_NUMBER ]] || [[ -z $Environment]]
then
  help
  exit 1
fi

echo "Find and replace values..."
sed -i "s/RELEASE/${Release}" ./Chart/Chart.yaml
sed -i "s/RELEASE/${Release}" ./Chart/values.yaml
sed -i "s/DRONE_BUILD_NUMBER/${DRONE_BUILD_NUMBER}" ./Chart/Chart.yaml
sed -i "s/DRONE_BUILD_NUMBER/${DRONE_BUILD_NUMBER}" ./Chart/values.yaml

echo "Packaging helm chart..."
helm package ./Chart/ --version $Release --app-version $DRONE_BUILD_NUMBER

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
git commit -m "Publishing KubeBackup ${Release}"
git push