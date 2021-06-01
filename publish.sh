#!/bin/sh

# Build and publish docker image to registry
#
docker build -t teonet -f ./Dockerfile ../.
docker tag teonet docker.pkg.github.com/kirill-scherba/teonet-go/teonet:$1
docker push docker.pkg.github.com/kirill-scherba/teonet-go/teonet:$1
