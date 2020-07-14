FROM golang:1.14 as builder

COPY . /opt
WORKDIR /opt

RUN CGO_ENABLED=0 go build -o /opt/bin/app github.com/certusone/near_exporter/cmd/near_exporter

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /opt/bin/app /

CMD ["/app"]
