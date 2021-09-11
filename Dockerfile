FROM ubuntu:focal AS builder

ARG LIBPCAP_VERSION="1.10.1"
ARG TELESCREEN_VERSION="unknown"
ARG TELESCREEN_REVISION="unknown"

ENV GOOS=linux
ENV GOARCH=amd64

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      git \
      wget \
      ca-certificates \
      golang \
      build-essential \
      bison \
      flex \
      libpcap-dev

WORKDIR /go/src/github.com/wide-vsix/telescreen
COPY . .

RUN wget https://www.tcpdump.org/release/libpcap-${LIBPCAP_VERSION}.tar.gz -O libpcap.tar.gz \
 && mkdir libpcap \
 && tar xzvf libpcap.tar.gz -C libpcap --strip-components 1 \
 && cd libpcap \
 && ./configure --with-pcap=linux \
 && make \
 && cd -

RUN go build \
      -tags netgo -installsuffix netgo \
      -ldflags "-s -w -linkmode external -extldflags -static -L ./libpcap -X 'main.VERSION=${TELESCREEN_VERSION}' -X 'main.REVISION=${TELESCREEN_REVISION}'" \
      -o bin/telescreen \
      -v \
      ./cmd/telescreen/main.go

FROM alpine:latest

WORKDIR /work
COPY --from=builder /go/src/github.com/wide-vsix/telescreen/bin/telescreen ./
ENTRYPOINT ["/work/telescreen"]
CMD ["--container"]
