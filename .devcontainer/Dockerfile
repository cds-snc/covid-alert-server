FROM ruby:2.6

ENV DEBIAN_FRONTEND=noninteractive

ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

ARG GO_VERSION=1.14.3
ENV GOROOT=/usr/local/go

RUN wget https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -xvf go${GO_VERSION}.linux-amd64.tar.gz \
    && mv go /usr/local \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

RUN export PATH=$GOPATH/bin:$GOROOT/bin:$PATH

RUN apt-get update \
    && apt-get -y install --no-install-recommends apt-utils dialog 2>&1 \
    && apt-get -y install protobuf-compiler git openssh-client less iproute2 procps lsb-release libsodium-dev mariadb-client \
    && gem install bundler \
    && gem install ruby-debug-ide \
    && gem install debase \
    && groupadd --gid $USER_GID $USERNAME \
    && useradd -s /bin/bash --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME\
    && chmod 0440 /etc/sudoers.d/$USERNAME \
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir /etc/aws-certs
RUN wget -P /etc/aws-certs https://s3.amazonaws.com/rds-downloads/rds-ca-2019-root.pem

ENV DEBIAN_FRONTEND=dialog

RUN echo "export PATH=${GOROOT}/bin:${PATH}" >> /root/.bashrc