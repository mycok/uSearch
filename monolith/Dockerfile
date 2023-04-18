########################################################
# STEP 1 use a temporary image to build a static binary.
########################################################
FROM golang:1.18 AS builder

# Pull build dependencies.
WORKDIR /src/
COPY . .
RUN make deps

# Build static image.
RUN GIT_SHA=$(git rev-parse --short HEAD) && \
CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
go build -a \
-ldflags "-extldflags '-static' -w -s -X main.appSHA=$GIT_SHA" \
-o /go/bin/usearch-monolith \
/src/

########################################################
# STEP 2 create alpine image with trusted certs
########################################################
FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /go/bin/uSearch-monolith /go/bin/usearch-monolith
ENTRYPOINT [ "/go/bin/uSearch-monolith" ]