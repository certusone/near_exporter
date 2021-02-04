FROM docker.io/golang:1.15@sha256:c161abf0cde3969e05f6914a86cab804b2b0df515f4ff9570475b25547ba7959 as builder

COPY . /opt
WORKDIR /opt

RUN CGO_ENABLED=0 go build -o /opt/bin/app github.com/certusone/near_exporter/cmd/near_exporter

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /opt/bin/app /

CMD ["/app"]
