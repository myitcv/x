FROM ubuntu:18.04

RUN apt-get -qq update
RUN apt-get -qq -y install sudo apt-utils git curl jq unzip gnupg2

RUN export go_bootstrap="$(curl -s https://golang.org/dl/?mode=json | jq -r '.[0] .version')" && \
  curl -sL https://dl.google.com/go/$go_bootstrap.linux-amd64.tar.gz | tar -C / -zx

ENV PATH=/go/bin:$PATH

RUN git clone -q https://github.com/myitcv/vbash /vbash/src/github.com/myitcv/vbash && \
  export GOPATH=/vbash && \
  go install github.com/myitcv/vbash

ARG USER
ARG UID
ARG DOCKER_WORKING_DIR
ARG CHROME_CHANNEL

RUN echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list && \
  curl -sL https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add - && \
  apt-get -qq update && \
  apt-get -qq -y install google-chrome-${CHROME_CHANNEL}

ENV PATH=/vbash/bin:$PATH
ENV GOPATH=/home/$USER/gopath

RUN groupadd -g $UID $USER && \
    adduser --uid $UID --gid $UID --disabled-password --gecos "" $USER

RUN sudo -u $USER mkdir -p $DOCKER_WORKING_DIR

# enable sudo
RUN usermod -aG sudo $USER
RUN echo "$USER ALL=(ALL:ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER

USER $USER
