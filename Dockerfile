FROM scratch

WORKDIR /app
COPY kubebackup /app/KubeBackup
ENTRYPOINT ["/app/KubeBackup"]
