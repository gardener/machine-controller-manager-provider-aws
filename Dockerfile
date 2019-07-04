FROM alpine:3.6

RUN apk add --update bash curl
COPY bin/rel/cmi-server /cmi-server
WORKDIR /
ENTRYPOINT ["/cmi-server"]
