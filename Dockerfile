#
# Install the latest versions of our mods.  This is done as a separate step
# so it will pull from an image cache if possible, unless there are changes.
#
FROM --platform=${BUILDPLATFORM} golang:1.15.6-alpine AS buildmod
ENV CGO_ENABLED=0
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download

#
# Compile the application.
#
FROM buildmod AS build
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /build/clusterizer clusterizer/clusterizer.go

#
# Base OS image for published images
#
FROM alpine AS base
RUN apk update && apk upgrade

#
# Build the image.  This should be a --target on docker build.
#
FROM base AS clusterizer-image
WORKDIR /app
COPY --from=build /build/clusterizer /app/clusterizer
CMD ["/app/clusterizer"]
