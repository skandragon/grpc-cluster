TARGETS=test local

PLATFORM=linux/amd64,linux/arm64

build=docker buildx build --pull --platform ${PLATFORM}

deps = \
	clusterizer/clusterizer.go

pb_deps = syncc/syncc.pb.go

now = `date -u +%Y%m%dT%H%M%S`

.PHONY: all
all: ${TARGETS}

.PHONY: local
local: bin/clusterizer

.PHONY: images
images: grpc-cluster-image

.PHONY: clean
clean:
	rm -f *.buildtime
	rm -f syncc/syncc.pb.go
	rm -f bin/*

syncc/syncc.pb.go: go.mod syncc/syncc.proto
	protoc --go_out=plugins=grpc:syncc syncc/syncc.proto

bin/clusterizer: go.mod syncc/syncc.pb.go clusterizer/clusterizer.go
	[ -d bin ] || mkdir bin
	go build -o bin/clusterizer clusterizer/clusterizer.go

.PHONY: grpc-cluster-image
grpc-cluster-image: grpc-cluster-image.buildtime

grpc-cluster-image.buildtime: ${deps} ${pb_deps} Dockerfile
	@${build} --tag docker.flame.org/library/grpc-cluster:latest --tag docker.flame.org/library/grpc-cluster:v${now} --target clusterizer-image . --push
	touch grpc-cluster-image.buildtime

.PHONY: test
test: ${pb_deps}
	go test  -race ./...
