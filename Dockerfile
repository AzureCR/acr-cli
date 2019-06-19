FROM golang:1.12.5-alpine AS gobuild-base
RUN apk add --no-cache \
	git \
	make

FROM gobuild-base AS acr-cli
WORKDIR /go/src/github.com/AzureCR/acr-cli
COPY . .
RUN make binaries && mv bin/acr /usr/bin/acr

FROM alpine:latest
COPY --from=acr-cli /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=acr-cli /usr/bin/acr /usr/bin/acr
ENTRYPOINT [ "acr" ]
