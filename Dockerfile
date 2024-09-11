FROM golang:1.23 as build

ENV CGO_ENABLED="0"
ENV GOOS="linux"
ENV GOARCH="amd64"

WORKDIR /kube/auth/server

COPY . ./

RUN go build -o /kube_server_bin ./

FROM scratch

USER 10001:10001

ENV PORT="6443"
ENV VERBOSE="enabled"

COPY --from=build /kube_server_bin /kube_server

ENTRYPOINT ["/kube_server"]