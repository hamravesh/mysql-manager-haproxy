FROM hub.hamdocker.ir/golang:1.23.3-bookworm AS gobuilder
WORKDIR /app 
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /tmp/mm-haproxy  main.go 

# TODO: add golang step 
FROM registry.hamdocker.ir/haproxy:3.1.0-bookworm
COPY --from=gobuilder /tmp/mm-haproxy /bin/mm-haproxy 
