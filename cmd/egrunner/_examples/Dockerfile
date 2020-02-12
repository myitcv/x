FROM golang

ARG UID
ARG GID

# create a user (gopher) with uid and gid to match the caller
RUN groupadd -g $GID gopher && \
    adduser --uid $UID --gid $GID --disabled-password --gecos "" gopher

# install envsubst
RUN apt-get update
RUN apt-get install -y gettext-base

USER gopher
