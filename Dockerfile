#FROM golang:1.23 as build
#
#ENV CGO_ENABLED=0
#
#WORKDIR /gobuild
#
#COPY go.mod go.sum ./
#RUN go mod download
#
#COPY . ./
#
#RUN go build -mod=readonly ./internal/...
#RUN go build -mod=readonly -v -o /taf-server ./cmd/server
#
FROM gcr.io/distroless/base:latest
# FROM golang:1.23
# FROM gcr.io/gcp-runtimes/ubuntu_16_0_4:latest

WORKDIR /taf
COPY testdata ./testdata
#COPY --from=build /taf-server .
COPY web ./web
COPY --chmod=0755 taf-server /taf/
ENTRYPOINT ["/taf/taf-server"]