from golang:alpine
WORKDIR /app
RUN apk add make 
COPY . .
RUN make build
CMD ["/app/nom", "--config-path", "docker-config.yml"]