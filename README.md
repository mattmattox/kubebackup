# What is KubeBackup?
KubeBackup is a tool for backing up the configuration files in a Kubernetes cluster and uploading them to a S3 bucket.

## How does KubeBackup work?
KubeBackup accessing the kubernetes API from inside a containter. Inside that containter there is a script will export all the cluster and namespace yaml files. These files can be used to redeploy an environment. All the exported yaml files are compressed and uploaded to an S3 Bucket.

## Setup
You must edit `secret.yaml` (remember to `base64` the values) to reflect your S3 details.
You can also edit the setting `CRON_SCHEDULE` if you want to change when the backup happens. Note: It uses the standard crontab formate.

## Deploy
kubectl apply -f secret.yaml
kubectl apply -f deployment.yaml
