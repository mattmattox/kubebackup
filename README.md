![KubeBackup Logo](https://github.com/mattmattox/kubebackup/raw/master/assets/kubebackup-logo.png)

[![Build Status](https://github.com/mattmattox/kubebackup/actions/workflows/build.yml/badge.svg)](https://github.com/mattmattox/kubebackup/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/username/kubebackup)](https://goreportcard.com/report/github.com/mattmattox/kubebackup)
![Docker Pulls](https://img.shields.io/docker/pulls/cube8021/kubebackup.svg)

# What is KubeBackup?

KubeBackup is a tool for backing up the configuration files in a Kubernetes cluster and uploading them to a S3 bucket.

## How does KubeBackup work?

KubeBackup accesses the Kubernetes API from inside a container. Inside that container, a script exports all the cluster and namespace yaml files. These files can be used to redeploy an environment. All the exported yaml files are compressed and uploaded to an S3 Bucket.

## Install

## Install / Upgrade
```
helm repo add SupportTools https://charts.support.tools
helm repo update
helm upgrade --install kubebackup SupportTools/kubebackup \
--set s3.region="us-east-2" \
--set s3.bucket="my-bucket" \
--set s3.folder="my-cluster" \
--set s3.accessKey="S3_ACCESS_KEY_GOES_HERE" \
--set s3.secretKey="S3_SECRET_KEY_GOES_HERE"
```


## How it works
KubeBackup is a helm chart that deploys a pod. This will take a YAML backup of your cluster and upload it to an S3 bucket.

The script connects to the Kubernetes API using either the provided kubeconfig file or the in-cluster configuration, if available. It then retrieves the list of available API resources and iterates through them to fetch namespaced and cluster-scoped objects.

Namespaced objects are grouped by namespace and saved in the `namespace-scoped/<namespace>/<object>` directory, while cluster-scoped objects are saved in the `cluster-scoped/<object>` directory. The output files are named <object-name>.yaml.

## Configuration
The following table lists the configurable parameters of the KubeBackup chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `cube8021/kubebackup` |
| `image.tag` | Image tag | `v1.1.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `s3.region` | AWS Region | `us-east-2` |
| `s3.bucket` | S3 Bucket | `kubebackup` |

## Building the script from source
To build the script from source, you will need to have the following installed:
* [Go](https://golang.org/dl/)
* [Docker](https://www.docker.com/get-started)

To build the script, run the following commands:
```
git clone
cd kubebackup
make build
```

## Contributing
If you would like to contribute to this project, please fork the repo and submit a pull request.

## License
This project is licensed under the Apache License - see the [LICENSE](LICENSE) file for details.