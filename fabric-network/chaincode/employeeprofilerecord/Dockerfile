# Chaincode-as-a-Service (CCaaS) image for employeeprofilerecord (CC-5).
# Runs as an independent, long-running server the peer dials OUT to —
# chosen specifically to avoid mounting the host Docker socket into peer
# containers. Dev-grade: no TLS between peer and this server (see main.go).
FROM golang:1.21 AS build
WORKDIR /src
COPY . .
RUN go build -o /out/employeeprofilerecord .

FROM debian:bookworm-slim
COPY --from=build /out/employeeprofilerecord /usr/local/bin/employeeprofilerecord
ENV CHAINCODE_SERVER_ADDRESS=0.0.0.0:9999
EXPOSE 9999
ENTRYPOINT ["/usr/local/bin/employeeprofilerecord"]
