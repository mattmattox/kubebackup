![KubeBackup Logo](https://github.com/mattmattox/kubebackup/raw/master/assets/kubebackup-logo.png)

[![Build Status](https://drone.support.tools/api/badges/mattmattox/kubebackup/status.svg)](https://drone.support.tools/mattmattox/kubebackup)
![Docker Pulls](https://img.shields.io/docker/pulls/cube8021/kubebackup.svg)

# What is KubeBackup?
KubeBackup is a tool for backing up the configuration files in a Kubernetes cluster and uploading them to a S3 bucket.

## How does KubeBackup work?
KubeBackup accessing the kubernetes API from inside a containter. Inside that containter there is a script will export all the cluster and namespace yaml files. These files can be used to redeploy an environment. All the exported yaml files are compressed and uploaded to an S3 Bucket.

## Install
```
helm repo add SupportTools https://charts.support.tools
helm repo update
helm install kubebackup SupportTools/kubebackup \
--set s3.region="us-east-2" \
--set s3.bucket="kubebackup" \
--set s3.accessKey="AWS_ACCESS_KEY_GOES_HERE" \
--set s3.secretKey="AWS_SECRET_KEY_GOES_HERE" \
--version v1.1.0
```

## How it works
KubeBackup is a helm chart that deploys a cronjob. The cronjob will run every 24 hours and export the cluster and namespace yaml files. The yaml files are compressed and uploaded to an S3 bucket.

The script connects to the Kubernetes API using either the provided kubeconfig file or the in-cluster configuration, if available. It then retrieves the list of available API resources and iterates through them to fetch namespaced and cluster-scoped objects.

Namespaced objects are grouped by namespace and saved in the namespace/<namespace>/<object> directory, while cluster-scoped objects are saved in the clusterobjects/<object> directory. The output files are named <object-name>.yaml.

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