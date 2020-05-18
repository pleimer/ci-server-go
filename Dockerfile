FROM golang
ENV D=/go/src/

ARG OC_LOC=https://mirror.openshift.com/pub/openshift-v4/clients/ocp/4.3.0/openshift-client-linux-4.3.0.tar.gz

WORKDIR $D
COPY . $D 

 RUN curl -sL https://deb.nodesource.com/setup_14.x | bash - &&\
    apt-get install nodejs && \
    npm install --global smee-client && \
    go build cmd/server.go && \
    curl -SL $OC_LOC |\
    tar -xvz -C /usr/bin --exclude="README.md"

EXPOSE 3000

ENTRYPOINT ./server
