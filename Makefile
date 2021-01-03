TARGETS=test grpc-cluster-image

PLATFORM=linux/amd64,linux/arm64

build=docker buildx build --pull --platform ${PLATFORM}

deps = \
	main.go

#pb_deps = go.mod tunnel/tunnel.proto

now = `date -u +%Y%m%dT%H%M%S`

.PHONY: all
all: ${TARGETS}

.PHONY: clean
clean:
	rm -f *.buildtime

#tunnel/tunnel.pb.go: ${pb_deps}
#	protoc --go_out=plugins=grpc:tunnel tunnel/tunnel.proto

.PHONY: grpc-cluster-image
grpc-cluster-image: grpc-cluster-image.buildtime

grpc-cluster-image.buildtime: ${deps}
	@${build} --tag docker.flame.org/library/grpc-cluster:latest --tag docker.flame.org/library/grpc-cluster:v${now} --target main-image . --push
	touch grpc-cluster-image.buildtime

.PHONY: test
test:
	echo "TODO: Add tests..."
	#go test  -race ./...
