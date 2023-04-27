FROM ubuntu:23.04 AS builder

ARG TARGETARCH

# Install build dependencies
RUN apt-get update && \
    apt-get install -y curl ca-certificates

# Download golang
RUN curl -s https://dl.google.com/go/go1.20.linux-${TARGETARCH}.tar.gz | tar -v -C /usr/local -xz
ENV PATH $PATH:/usr/local/go/bin

# Build server
WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Build
COPY . .
RUN go build -v -o /out/server ./cmd/main.go

FROM ubuntu:23.04

# Copy build output
COPY --from=builder /out/server /app/server

WORKDIR /app
CMD ["/app/server"]
