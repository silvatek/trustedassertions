FROM gcr.io/distroless/base:latest

WORKDIR /taf
COPY testdata ./testdata
COPY web ./web
COPY --chmod=0755 taf-server /taf/
ENTRYPOINT ["/taf/taf-server"]