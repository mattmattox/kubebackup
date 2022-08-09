FROM alpine:latest

##Installing Packages
RUN sed -i 's/http\:\/\/dl-cdn.alpinelinux.org/https\:\/\/alpine.global.ssl.fastly.net/g' /etc/apk/repositories && \
apk update && apk add \
bash \
curl \
grep \
sed \
aws-cli

##Install kubectl
RUN curl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x /usr/local/bin/kubectl

ADD *.sh /
RUN chmod +x /*.sh

ENTRYPOINT ["/start.sh"]