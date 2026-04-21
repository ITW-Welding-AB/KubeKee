FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# LDFLAGS can be injected at build time:
#   docker build --build-arg LDFLAGS="-X github.com/ITW-Welding-AB/KubeKee/internal/cli.version=v1.2.3 -s -w" .
ARG LDFLAGS="-s -w"
RUN CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o kubekee ./cmd/kubekee

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/kubekee /usr/local/bin/kubekee
ENTRYPOINT ["kubekee"]

