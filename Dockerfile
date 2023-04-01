FROM golang:1.17-alpine

WORKDIR /app

COPY . .

RUN apk add --no-cache ca-certificates && \
    chmod +x ./KubeBackup

ENTRYPOINT ["./KubeBackup"]
