FROM ubuntu:18.04

RUN apt-get -qq update
RUN apt-get -qq -y install sudo apt-utils git curl jq unzip gnupg2 python python-pip

RUN export go_bootstrap="$(curl -s https://golang.org/dl/?mode=json | jq -r '.[0] .version')" && \
  curl -sL https://dl.google.com/go/$go_bootstrap.linux-amd64.tar.gz | tar -C / -zx

ENV PATH=/go/bin:$PATH

RUN git clone -q https://github.com/myitcv/vbash /vbash/src/github.com/myitcv/vbash && \
  export GOPATH=/vbash && \
  go install github.com/myitcv/vbash

ARG CHROME_CHANNEL

RUN pip install awscli

ENV PATH=/vbash/bin:$PATH

RUN echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list && \
  curl -sL https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add - && \
  apt-get -qq update && \
  apt-get -qq -y install google-chrome-${CHROME_CHANNEL}

ENV NODE_VERSION=v10.10.0
ENV NPM_VERSION=v6.4.1
ENV NVM_VERSION=v0.33.11
ENV NVM_DIR=/nvm
ENV PATH=$NVM_DIR/versions/node/$NODE_VERSION/bin:$PATH
RUN git config --global advice.detachedHead false

RUN git clone -q --branch $NVM_VERSION https://github.com/creationix/nvm.git $NVM_DIR \
  && . $NVM_DIR/nvm.sh \
  && nvm install $NODE_VERSION > /dev/null \
  && npm install -g npm@$NPM_VERSION
