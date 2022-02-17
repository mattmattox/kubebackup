#!/bin/ash

set -e

echo "Creating kubeconfig..."
if [ -z ${PLUGIN_NAMESPACE} ]; then
  PLUGIN_NAMESPACE="default"
fi
echo "PLUGIN_NAMESPACE: ${PLUGIN_NAMESPACE}"

if [ -z ${PLUGIN_KUBERNETES_USER} ]; then
  PLUGIN_KUBERNETES_USER="default"
fi
echo "PLUGIN_KUBERNETES_USER: ${PLUGIN_KUBERNETES_USER}"

if [ ! -z ${PLUGIN_KUBERNETES_TOKEN} ]; then
  KUBERNETES_TOKEN=${PLUGIN_KUBERNETES_TOKEN}
fi
echo "KUBERNETES_TOKEN: ${KUBERNETES_TOKEN}"

if [ ! -z ${PLUGIN_KUBERNETES_SERVER} ]; then
  KUBERNETES_SERVER=${PLUGIN_KUBERNETES_SERVER}
fi
echo "KUBERNETES_SERVER: ${KUBERNETES_SERVER}"

if [ ! -z ${PLUGIN_KUBERNETES_CERT} ]; then
  KUBERNETES_CERT=${PLUGIN_KUBERNETES_CERT}
fi
echo "KUBERNETES_CERT: ${KUBERNETES_CERT}"

if [ ! -z ${KUBERNETES_CERT} ]; then
  echo ${KUBERNETES_CERT} | base64 -d > ca.crt
  kubectl config set-cluster default --server=${KUBERNETES_SERVER} --certificate-authority=ca.crt
else
  echo "WARNING: Using insecure connection to cluster"
  kubectl config set-cluster default --server=${KUBERNETES_SERVER} --insecure-skip-tls-verify=true
fi
kubectl config set-context default --cluster=default --user=${PLUGIN_KUBERNETES_USER}
kubectl config use-context default
kubectl cluster-info

echo "Starting backup..."
DATE="$(date '+%Y%m%d%H%M%S')"
echo "DATE: ${DATE}"

rm -rf /tmp/backup_files
mkdir -p /tmp/backup_files
cd /tmp/backup_files

echo "Dumping namespaced scoped objects..."
objects=`kubectl api-resources --verbs=list --namespaced -o name`
for namespace in `kubectl get ns -o name | awk -F '/' '{print $2}'`
do
  echo "Namespace: $namespace"
  mkdir -p namespace/"$namespace"
  for object in $objects
  do
    mkdir -p namespace/"$namespace"/"$object"
    echo "Object: $object"
    for item in `kubectl -n $namespace get $object -o name | awk -F '/' '{print $2}'`
    do
      echo "item: $item"
      kubectl -n $namespace get $object $item -o yaml > namespace/"$namespace"/"$object"/"$item".yaml
    done
  done
done

echo "Dumping cluster scoped objects..."
objects=`kubectl api-resources --verbs=list --cluster-scoped -o name`
mkdir -p clusterobjects
for clusterobject in $clusterobjects
do
  echo "Cluster Object: $clusterobject"
  mkdir -p clusterobjects/"$clusterobject"
  for clusteritem in `kubectl get $clusterobject -o name | awk -F '/' '{print $2}'`
  do
    echo "Cluster Item: $clusteritem"
    kubectl get $clusterobject $clusteritem -o yaml > clusterobjects/"$clusterobject"/"$clusteritem".yaml
  done
done

mkdir -p /backup/
cd /backup/
echo "Running tar..."
tar -czvf "$DATE".tar.gz -C /tmp/backup_files .
echo "Done."

echo "Starting sync to S3..."
if [[ ! -z $CLUSTER ]]
then
  aws s3 sync /backup s3://"$S3_BUCKET"/"$CLUSTER"
else
  aws s3 sync /backup s3://"$S3_BUCKET"
fi
echo "$(date) End"
