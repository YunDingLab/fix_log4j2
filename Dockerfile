FROM golang:1.15 AS building

ENV CGO_ENABLED=0

COPY . /go/src/fix_log4j2

WORKDIR /go/src/fix_log4j2
RUN make local

FROM alpine:edge

COPY --from=building /go/src/fix_log4j2/bundles/fix_log4j2 /usr/local/bin/fix_log4j2
COPY --from=building /go/src/fix_log4j2/internal/config/example.yaml /fix_log4j2.yaml

CMD [ "/usr/local/bin/fix_log4j2", "-c", "/fix_log4j2.yaml"]
