#!/bin/ash

set -e

echo "$(date) - Start"
echo "Starting export..."
echo "Starting backup..."
DATE="$(date '+%Y%m%d%H%M%S')"
echo "Date: $DATE"

rm -rf /tmp/backup_files
mkdir -p /tmp/backup_files
cd /tmp/backup_files

mkdir namespaces
cd namespaces
for namespace in $(kubectl get namespaces -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
do
	echo "Namespace: $namespace"
	mkdir -p "$namespace"
	namespaceresources="certificatesigningrequests componentstatuses configmaps controllerrevisions cronjobs customresourcedefinition daemonsets deployments endpoints events horizontalpodautoscalers ingresses jobs limitranges networkpolicies persistentvolumeclaims persistentvolumes poddisruptionbudgets pods podsecuritypolicies podtemplates replicasets replicationcontrollers resourcequotas rolebindings roles secrets serviceaccounts services statefulsets storageclasses"
  for namespaceresource in $namespaceresources
	do
		echo "Resource: $namespaceresource"
    ##Getting RAW yaml output
		kubectl get "$namespaceresource" -n "$namespace" -o yaml > ./"$namespace"/"$namespaceresource"-raw.yaml
    ##Filtering output Rancher metadata
    cat ./"$namespace"/"$namespaceresource"-raw.yaml | \
    grep -v 'cattle.io/timestamp:' | \
    grep -v 'cni.projectcalico.org/podIP:' | \
    grep -v 'creationTimestamp:' | \
    grep -v 'uid:' | \
    grep -v 'resourceVersion:' | \
    grep -v 'selfLink:' | \
    sed '/^status:/q' | \
    grep -v 'status:' > ./"$namespace"/"$namespaceresource"-generic.yaml
	done
done

mkdir -p /backup_data/
cd /backup_data/
echo "Running tar..."
tar -czvf "$DATE".tar.gz -C /tmp/backup_files .
echo "Starting sync to S3..."
if [[ ! -z $CLUSTER ]]
then
	aws s3 sync /backup_data s3://"$S3_BUCKET"/"$CLUSTER"
else
	aws s3 sync /backup_data s3://"$S3_BUCKET"
fi
echo "$(date) End"
