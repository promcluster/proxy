FROM golang:1.17 as build-env

COPY . /root

WORKDIR /root

RUN \
  export GO111MODULE=on &&\
  cd cmd/proxy &&\
  go build -v -mod=vendor -o /promcluster-proxy

FROM debian:9
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=build-env /promcluster-proxy /usr/bin/promcluster-proxy
COPY --from=build-env /root/config/default.yaml /etc/promcluster-proxy.yaml

CMD ["/usr/bin/promcluster-proxy", "-config=/etc/promcluster-proxy.yaml"]
