FROM docker.io/golang:alpine AS builder
RUN apk add make alpine-sdk gcc

WORKDIR /app

# Install dependencies
COPY go.* ./
RUN go mod download

# Build nom
COPY . .
ENV CGO_ENABLED=1
RUN make build


FROM docker.io/golang:alpine
RUN apk add nano
COPY --from=builder /app/nom /usr/local/bin/

WORKDIR /app
COPY docker-config.yml ./
ENTRYPOINT ["nom"]
CMD ["--config-path", "docker-config.yml"]
