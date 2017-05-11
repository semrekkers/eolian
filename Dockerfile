FROM golang:1.8.1
MAINTAINER brett@buddin.us

RUN \
    apt-get update -y && \
    apt-get install -y \
        build-essential \
        libportmidi-dev \
        portaudio19-dev \
    && rm -rf /var/lib/apt/lists/*

COPY . /go/src/buddin.us/eolian
WORKDIR /go/src/buddin.us/eolian
