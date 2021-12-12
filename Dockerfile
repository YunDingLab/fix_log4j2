FROM golang:1.15 AS building

ENV CGO_ENABLED=0

COPY . /go/src/fix_log4j

WORKDIR /go/src/fix_log4j
RUN make local

FROM alpine:edge

COPY --from=building /go/src/fix_log4j/bundles/fix_log4j /usr/local/bin/fix_log4j
COPY --from=building /go/src/fix_log4j/internal/config/example.yaml /fix_log4j.yaml

CMD [ "/usr/local/bin/fix_log4j", "-c", "/fix_log4j.yaml"]
