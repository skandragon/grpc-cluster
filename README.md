# grpc-cluster

This is an example of a simple GRPC server which uses a Kubernetes headless
service to discover and maintain a connection to all other running instances.

This is not a robust solution to the problem of maintaining shared state,
but is an experiment in how to use a headless service to enumerate pods,
and track them as they come and go.

This implementation makes sure to avoid connecting to itself, using
an injected environment variable to provide the list of addresses.

It's not clear if IPv4 and IPv6 are both added to the headless service for
the same pod, or if at most one address is added.  For now, I think
it's up to the code to handle this situation and de-dupelicate connections
in some way.
