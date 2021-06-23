FROM alpine:latest

## Switching mirror
RUN sed -i 's/http\:\/\/dl-cdn.alpinelinux.org/http\:\/\/mirror.clarkson.edu/g' /etc/apk/repositories

##Installing Packages
RUN apk --no-cache add \
bash \
curl \
py-pip \
grep \
sed \
python &&\
pip install --upgrade \
pip \
awscli

##Install kubectl
RUN curl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x /usr/local/bin/kubectl

ADD *.sh /
RUN chmod +x /*.sh

ENTRYPOINT ["/start.sh"]
