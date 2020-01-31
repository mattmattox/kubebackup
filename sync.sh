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

echo "################################################################################################################"
echo "::Cluster Info::"
mkdir clusterinfo
clusterresources="certificatesigningrequests clusterrolebindings clusterroles componentstatuses configmaps controllerrevisions cronjobs customresourcedefinition daemonsets deployments endpoints events horizontalpodautoscalers ingresses jobs limitranges namespaces networkpolicies nodes persistentvolumeclaims persistentvolumes poddisruptionbudgets pods podsecuritypolicies podtemplates replicasets replicationcontrollers resourcequotas rolebindings roles secrets serviceaccounts services statefulsets storageclasses"
for clusterresource in $clusterresources
do
	echo "Resource: $clusterresource"
	kubectl get "$clusterresource" -o yaml > ./clusterinfo/"$clusterresource"
done
echo "################################################################################################################"

echo "################################################################################################################"
echo "::Namespace Info::"
mkdir namespaces
cd namespaces
for namespace in $(kubectl get namespaces -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
do
	echo "################################################################################################################"
	echo "Namespace: $namespace"
	mkdir -p "$namespace"
	namespaceresources="certificatesigningrequests componentstatuses configmaps controllerrevisions cronjobs customresourcedefinition daemonsets deployments endpoints events horizontalpodautoscalers ingresses jobs limitranges networkpolicies persistentvolumeclaims persistentvolumes poddisruptionbudgets pods podsecuritypolicies podtemplates replicasets replicationcontrollers resourcequotas rolebindings roles secrets serviceaccounts services statefulsets storageclasses"
	for namespaceresource in $namespaceresources
	do
		echo "Resource: $namespaceresource"
		kubectl get "$namespaceresource" -n "$namespace" -o yaml > ./"$namespace"/"$namespaceresource"
		if [[ "$namespaceresource" == "secret" ]]
		then
			/kubedecode "$namespaceresource" "$namespace" > ./"$namespace"/secret_decoded
		fi
	done
	echo "################################################################################################################"
done
echo "################################################################################################################"

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
