#!/bin/sh

socat TCP-LISTEN:50051,fork TCP:docker.for.mac.localhost:50051 &
socat TCP-LISTEN:50052,fork TCP:docker.for.mac.localhost:50052 &
socat TCP-LISTEN:50061,fork TCP:docker.for.mac.localhost:50061 &
socat TCP-LISTEN:50071,fork TCP:docker.for.mac.localhost:50071 &

exec "$@"
