FROM golang:alpine
WORKDIR /app
RUN apk add make alpine-sdk gcc
COPY . .
ENV CGO_ENABLED=1
RUN make build
CMD ["/app/nom", "--config-path", "docker-config.yml"]
