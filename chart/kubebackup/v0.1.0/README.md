# KubeBackup

[KubeBackup](https://github.com/mattmattox/kubebackup) is a tool for backing up the configuration files in a Kubernetes cluster and uploading them to a S3 bucket.

## TL;DR;

```console
$ helm install stable/kubebackup
```

### How does KubeBackup work?

KubeBackup accessing the kubernetes API from inside a containter. Inside that containter there is a script will export all the cluster and namespace yaml files. These files can be used to redeploy an environment. All the exported yaml files are compressed and uploaded to an S3 Bucket.

## Prerequisites

- Kubernetes 1.4+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm install --name my-release stable/kubebackup
```

The command deploys kubebackup on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.
