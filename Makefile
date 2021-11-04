all:
#	go get -d -v ./...
	go install -ldflags "-s -w" ./cmd/*

# To build in docker envrionment you can run following cmd directly
docker:
	sudo docker build --force-rm -t k8s-rdt-controller:0.1 .

clean:
	rm -f `go env GOPATH`/bin/agent `go env GOPATH`/bin/server
