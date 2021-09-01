FROM ubuntu:focal AS builder

ARG LIBPCAP_VERSION="1.10.1"
ARG INTERCEPTOR_VERSION="unknown"
ARG INTERCEPTOR_REVISION="unknown"

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

WORKDIR /go/src/github.com/wide-vsix/dns-query-interceptor
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
      -ldflags "-s -w -linkmode external -extldflags -static -L ./libpcap -X 'main.VERSION=${INTERCEPTOR_VERSION}' -X 'main.REVISION=${INTERCEPTOR_REVISION}'" \
      -o bin/interceptor \
      -v \
      ./cmd/interceptor/main.go

FROM alpine:latest

WORKDIR /work
COPY --from=builder /go/src/github.com/wide-vsix/dns-query-interceptor/bin/interceptor ./
ENTRYPOINT ["/work/interceptor"]
