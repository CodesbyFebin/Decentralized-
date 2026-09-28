FROM golang:1.26.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/dh-control ./cmd/dh-control

FROM alpine:3.22
RUN addgroup -S dh && adduser -S -G dh -u 10001 dh
USER 10001:10001
COPY --from=build /out/dh-control /usr/local/bin/dh-control
ENTRYPOINT ["/usr/local/bin/dh-control"]
