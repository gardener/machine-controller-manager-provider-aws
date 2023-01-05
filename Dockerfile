#############      builder                          #############
FROM golang:1.19.4 AS builder

WORKDIR /go/src/github.com/gardener/machine-controller-manager-provider-aws
COPY . .

RUN .ci/build

#############      machine-controller               #############
FROM gcr.io/distroless/static-debian11:nonroot AS machine-controller
WORKDIR /

COPY --from=builder /go/src/github.com/gardener/machine-controller-manager-provider-aws/bin/rel/machine-controller /machine-controller
ENTRYPOINT ["/machine-controller"]
