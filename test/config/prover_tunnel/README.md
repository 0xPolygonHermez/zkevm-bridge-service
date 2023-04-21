# Prover Tunnel

This tunnel runs tcp proxies pointing to the Docker host for the ports exposed by the [zkevm-prover](https://hub.docker.com/r/hermeznetwork/zkevm-prover).
This is necessary for ARM developers because the prover does not support this architecture at this point, so ARM developers need to tunnel to an external prover.

The Makefile will build this image with the same tag as the prover, so the docker-compose will run this fake prover instead.
