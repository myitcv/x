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

