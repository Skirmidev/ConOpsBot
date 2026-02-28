FROM cgr.dev/chainguard/go AS builder
WORKDIR /app
ADD . /app
RUN cd /app && go build ./cmd/main.go

FROM cgr.dev/chainguard/glibc-dynamic
WORKDIR /app
COPY --from=builder /app/main /app
ENTRYPOINT ["./main"]