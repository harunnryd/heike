# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
  go build -trimpath -ldflags="-s -w" -o /out/heike ./cmd/heike

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/heike /usr/local/bin/heike

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/heike"]
CMD ["daemon"]
