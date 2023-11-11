# Copyright 2021 Kirill Scherba <kirill@scherba.ru>.  All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
#
# Teonet docker file
#
# Build:
#
#  docker build -t teonet .
#
# Publish to github:
#
#  docker login docker.pkg.github.com -u USERNAME -p TOKEN
#  docker tag teonet docker.pkg.github.com/teonet-go/teonet/teonet:x.x.x
#  docker push docker.pkg.github.com/teonet-go/teonet/teonet:x.x.x
#
# Publish to local repository:
#
#  docker tag teonet 192.168.106.5:5000/teonet
#  docker push 192.168.106.5:5000/teonet
#
# Run docker container:
#
#  docker run --rm -it teonet
#
# Run in swarm claster:
#
#  docker volume create teonet-config
#  docker service create --constraint 'node.hostname == teonet'   --network teo-overlay --hostname=teo-go-01 --name teo-go-01 --mount type=volume,source=teonet-config,target=/root/.config/teonet 192.168.106.5:5000/teonet-go teonet -a 5.63.158.100 -r 9010 -n teonet teo-go-01
#  docker service create --constraint 'node.hostname == dev-ks-2' --network teo-overlay --hostname=teo-go-02 --name teo-go-02 --mount type=volume,source=teonet-config,target=/root/.config/teonet 192.168.106.5:5000/teonet-go teonet -a 5.63.158.100 -r 9010 -n teonet teo-go-02
#
# Or update existing service in swarm claster:
#
#  docker service update --image 192.168.106.5:5000/teonet-go teonet-go
#


# Docker builder
#
# 
#
# Docker build (included private repositories):
#
#   docker build --build-arg github_user="USERNAME" --build-arg github_personal_token="TOKEN_FOR_REPOSITORIES" -t teonet -f ./Dockerfile .
#
# Docker test run:
# it recomendet use host network when run teonet server application
#
#   docker tag teonet docker.pkg.github.com/kirill-scherba/teonet-go/teonet:x.x.x
#   docker push docker.pkg.github.com/kirill-scherba/teonet-go/teonet:x.x.x
#   docker run --restart always -it --name teonet --network host docker.pkg.github.com/kirill-scherba/teonet-go/teonet:x.x.x teonet -u
#
#   docker run --rm -it --name teonet-v4 -v $HOME/.config/teonet:/root/.config/teonet docker.pkg.github.com/kirill-scherba/teonet-go/teonet:x.x.x teonet -u -app-short teonet-v4-1 -send-to dBTgSEHoZ3XXsOqjSkOTINMARqGxHaXIDxl
#
#

# Docker builder
# 
FROM golang:1.21.3 AS builder

WORKDIR /go/src/github.com/teonet-go/teonet
# RUN apt update 

COPY ./ ./

# Add the keys
ARG github_user
ENV github_user=$github_user
ARG github_personal_token
ENV github_personal_token=$github_personal_token

# Select private repo
# RUN go env -w GOPRIVATE=\
# github.com/teonet-go/teonet,\
# github.com/teonet-go/teomon,\
# github.com/teonet-go/tru

# Change github url
RUN git config \
    --global \
    url."https://${github_user}:${github_personal_token}@github.com".insteadOf \
    "https://github.com"

RUN go get 
RUN go install ./cmd/teoapi 
RUN go install ./cmd/teoapicli
RUN go install ./cmd/teoecho 
RUN go install ./cmd/teonet

RUN ls /go/bin

CMD ["teonet"]

# #############################################################
# Compose production image
#
FROM ubuntu:latest AS production
WORKDIR /app

# runtime dependencies
# RUN apt update 

# install ssl cretificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# install previously built application
COPY --from=builder /go/bin/* /usr/local/bin/

CMD ["/usr/local/bin/teonet"]  
