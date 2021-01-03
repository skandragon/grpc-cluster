# grpc-cluster

This is an example of a simple GRPC server which uses a Kubernetes headless
service to discover and maintain a connection to all other running instances.

This is not a robust solution to the problem of maintaining shared state,
but is an experiment in 