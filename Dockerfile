FROM ubuntu:18.04

RUN apt-get -qq update
RUN apt-get -qq -y install sudo apt-utils git curl jq unzip gnupg2 python python-pip lsb-release

RUN mkdir /gobootstrap && export go_bootstrap="$(curl -s https://golang.org/dl/?mode=json | jq -r '.[0] .version')" && \
  curl -sL https://dl.google.com/go/$go_bootstrap.linux-amd64.tar.gz | tar --strip-components=1 -C /gobootstrap -zx

ENV PATH=/go/bin:$PATH

# Install awscli
RUN pip install awscli

# Install Node
ENV NODE_VERSION=v10.15.0
ENV NPM_VERSION=v6.5.0
ENV NVM_VERSION=v0.33.11
ENV NVM_DIR=/nvm
ENV PATH=$NVM_DIR/versions/node/$NODE_VERSION/bin:$PATH
RUN git config --global advice.detachedHead false

RUN git clone -q --branch $NVM_VERSION https://github.com/creationix/nvm.git $NVM_DIR \
  && . $NVM_DIR/nvm.sh \
  && nvm install $NODE_VERSION > /dev/null \
  && npm install -g npm@$NPM_VERSION

# Install Docker
RUN apt-get update
RUN apt-get install -y apt-transport-https ca-certificates curl gnupg2 software-properties-common
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
RUN apt-get update
RUN apt-get install -y docker-ce

# Install vbash
ARG VBASHPATH
ARG GOBINPATH
COPY $VBASHPATH /usr/bin/
COPY $GOBINPATH /usr/bin/

# Install protobuf
ARG PROTOBUF_VERSION
RUN mkdir /protobuf \
  && cd /protobuf \
  && curl -sL -o protobuf.zip https://github.com/google/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-linux-x86_64.zip \
  && unzip -q protobuf.zip

ENV PROTOBUF_INCLUDE=/protobuf/include

# Install Go
ARG GO_VERSION
RUN curl -sL https://dl.google.com/go/${GO_VERSION}.linux-amd64.tar.gz | tar -C / -zx
ENV PATH=/go/bin:$PATH

# Install Chrome
ARG CHROME_VERSION
ARG CHROME_CHANNEL
RUN echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list && \
  curl -sL https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - && \
  apt -qq -y update && \
  apt-get -qq -y install google-chrome-${CHROME_CHANNEL}

# Install chromedriver
ARG CHROMEDRIVER_77_VERSION
ARG CHROMEDRIVER_76_VERSION
ARG CHROMEDRIVER_75_VERSION
RUN mkdir /usr/bin/chromedriver

RUN bash -c '([[ "$(google-chrome --version)" =~ [[:space:]]77\. ]] && curl -s -o /usr/bin/chromedriver/chrome_driver.zip https://chromedriver.storage.googleapis.com/$CHROMEDRIVER_77_VERSION/chromedriver_linux64.zip) || true'
RUN bash -c '([[ "$(google-chrome --version)" =~ [[:space:]]76\. ]] && curl -s -o /usr/bin/chromedriver/chrome_driver.zip https://chromedriver.storage.googleapis.com/$CHROMEDRIVER_76_VERSION/chromedriver_linux64.zip) || true'
RUN bash -c '([[ "$(google-chrome --version)" =~ [[:space:]]75\. ]] && curl -s -o /usr/bin/chromedriver/chrome_driver.zip https://chromedriver.storage.googleapis.com/$CHROMEDRIVER_75_VERSION/chromedriver_linux64.zip) || true'

# Catch unknown versions
RUN ls /usr/bin/chromedriver/chrome_driver.zip

RUN cd /usr/bin/chromedriver && unzip -q chrome_driver.zip && chmod 755 chromedriver

ENV PATH=/usr/bin/chromedriver:$PATH
