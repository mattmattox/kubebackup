FROM alpine:3.3
MAINTAINER Matthew Mattox - matthew.mattox@rancher.com

##Install python and awscli
RUN apk --no-cache add \
	py-pip \
	python &&\
	pip install --upgrade \
	pip \
	awscli

##Install bash
RUN apk add --no-cache bash

##Install nano
RUN apk add --no-cache nano

##Install curl
RUN apk add --no-cache curl

##Install kubectl
RUN curl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x /usr/local/bin/kubectl

##Verify kubectl is working
RUN kubectl version --client

ENV KEY=,SECRET=,REGION=,BUCKET=,BUCKET_PATH=/,CRON_SCHEDULE="00 * * * *",PARAMS=

VOLUME ["/backup_data"]

ADD kubedecode /
ADD *.sh /
RUN chmod +x /*.sh

ENTRYPOINT ["/start.sh"]
#CMD [""]
#CMD exec /bin/sh -c "trap : TERM INT; (while true; do sleep 1000; done) & wait"
