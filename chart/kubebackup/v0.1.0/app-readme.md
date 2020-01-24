# KubeBackup
[KubeBackup](https://github.com/mattmattox/kubebackup) is a tool for backing up the configuration files in a Kubernetes cluster and uploading them to a S3 bucket.

## How does KubeBackup work?

KubeBackup accessing the kubernetes API from inside a containter. Inside that containter there is a script will export all the cluster and namespace yaml files. These files can be used to redeploy an environment. All the exported yaml files are compressed and uploaded to an S3 Bucket.
