TARGETS=test grpc-cluster-image

PLATFORM=linux/amd64,linux/arm64

build=docker buildx build --pull --platform ${PLATFORM}

deps = \
	clusterizer/clusterizer.go

pb_deps = syncc/syncc.pb.go

now = `date -u +%Y%m%dT%H%M%S`

.PHONY: all
all: ${TARGETS}

.PHONY: clean
clean:
	rm -f *.buildtime
	rm -f syncc/syncc.pb.go

syncc/syncc.pb.go: go.mod syncc/syncc.proto
	protoc --go_out=plugins=grpc:syncc syncc/syncc.proto

.PHONY: grpc-cluster-image
grpc-cluster-image: grpc-cluster-image.buildtime

grpc-cluster-image.buildtime: ${deps} ${pb_deps} Dockerfile
	@${build} --tag docker.flame.org/library/grpc-cluster:latest --tag docker.flame.org/library/grpc-cluster:v${now} --target clusterizer-image . --push
	touch grpc-cluster-image.buildtime

.PHONY: test
test: ${pb_deps}
	go test  -race ./...
