![KubeBackup Logo](https://github.com/mattmattox/kubebackup/raw/master/assets/kubebackup-logo.png)

![Docker Pulls](https://img.shields.io/docker/pulls/cube8021/kubebackup.svg)

# What is KubeBackup?
KubeBackup is a tool for backing up the configuration files in a Kubernetes cluster and uploading them to a S3 bucket.

## How does KubeBackup work?
KubeBackup accessing the kubernetes API from inside a containter. Inside that containter there is a script will export all the cluster and namespace yaml files. These files can be used to redeploy an environment. All the exported yaml files are compressed and uploaded to an S3 Bucket.

## Install
```
helm repo add kubebackup https://mattmattox.github.io/helm-chart/
helm install kubebackup kubebackup \
--set s3.region="us-east-2" \
--set s3.bucket="kubebackup" \
--set s3.accessKey="AWS_ACCESS_KEY_GOES_HERE" \
--set s3.secretKey="AWS_SECRET_KEY_GOES_HERE"
```
