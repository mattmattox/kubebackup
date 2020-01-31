FROM alpine:3.9
MAINTAINER Matthew Mattox - matthew.mattox@rancher.com

##Install python and awscli
RUN apk --no-cache add \
	py-pip \
	python &&\
	pip install --upgrade \
	pip \
	bash \
	curl \
	awscli

##Install kubectl
RUN curl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x /usr/local/bin/kubectl

VOLUME ["/backup_data"]

ADD kubedecode /
ADD *.sh /
RUN chmod +x /*.sh

ENTRYPOINT ["/start.sh"]
