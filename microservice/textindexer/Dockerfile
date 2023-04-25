########################################################
# STEP 1 use a temporary image to build a static binary.
########################################################
FROM golang:1.20-alpine3.17 AS builder

RUN apk add --no-cache git && \
apk add --no-cache bash

# Pull build dependencies.
WORKDIR /src/
COPY . .

# Build static image.
RUN GIT_SHA=$(git rev-parse --short HEAD) && \
CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
go build -a \
-ldflags "-extldflags '-static' -w -s -X main.appSHA=$GIT_SHA" \
-o /bin/usearch-textindexer \
/src/microservice/textindexer

########################################################
# STEP 2 create alpine image with trusted certs
########################################################
FROM alpine:3.17
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /bin/usearch-textindexer /bin/usearch-textindexer
ENTRYPOINT [ "/bin/usearch-textindexer" ]