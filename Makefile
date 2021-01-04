TARGETS=test local
PLATFORM=linux/amd64,linux/arm64
BUILD=docker buildx build --pull --platform ${PLATFORM}

deps = \
	clusterizer/clusterizer.go

pb_deps = syncc/syncc.pb.go

now = `date -u +%Y%m%dT%H%M%S`

#
# Default target.
#

.PHONY: all
all: ${TARGETS}

#
# Common components, like GRPC client code generation.
#

syncc/syncc.pb.go: go.mod syncc/syncc.proto
	protoc --go_out=plugins=grpc:syncc syncc/syncc.proto

#
# Build locally, mostly for development speed.
#

.PHONY: local
local: bin/clusterizer

bin/clusterizer: go.mod syncc/syncc.pb.go clusterizer/clusterizer.go
	[ -d bin ] || mkdir bin
	go build -o bin/clusterizer clusterizer/clusterizer.go

#
# Image builds
#

.PHONY: images
images: grpc-cluster-image

.PHONY: grpc-cluster-image
grpc-cluster-image: grpc-cluster-image.buildtime

grpc-cluster-image.buildtime: ${deps} ${pb_deps} Dockerfile
	@${BUILD} --tag docker.flame.org/library/grpc-cluster:latest --tag docker.flame.org/library/grpc-cluster:v${now} --target clusterizer-image . --push
	touch grpc-cluster-image.buildtime

#
# Test targets
#

.PHONY: test
test: ${pb_deps}
	go test  -race ./...

#
# Clean the world.
#

.PHONY: clean
clean:
	rm -f *.buildtime
	rm -f syncc/syncc.pb.go
	rm -f bin/*
