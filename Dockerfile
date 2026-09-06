FROM gcr.io/distroless/base:latest

ARG BUILD_TIME=
ENV BUILD_TIME=$BUILD_TIME

WORKDIR /taf
COPY testdata ./testdata
COPY web ./web
COPY --chmod=0755 taf-server /taf/
ENTRYPOINT ["/taf/taf-server"]