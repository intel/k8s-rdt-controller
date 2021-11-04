# Build k8s-rdt-controller
FROM golang:1.13.8 as builder

WORKDIR /go/src/k8s-rdt-controller
COPY . /go/src/k8s-rdt-controller

# use "go get" instead of "go mod" to bypass client-go dependence issue
# use build.sh to bypass error: "The command '/bin/sh -c go get -d -v ./...' returned a non-zero code: 1"
RUN ./build.sh

# Create k8s-rdt-controller image
FROM centos:7

COPY --from=builder /go/bin/agent /usr/bin/
COPY --from=builder /go/src/k8s-rdt-controller/scripts  /usr/bin/scripts/

RUN yum install -y epel-release
RUN yum install -y msr-tools


