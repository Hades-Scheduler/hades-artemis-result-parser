# Use an official Go runtime as a parent image
FROM golang:1.26-alpine@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468 AS builder

# Set the working directory inside the container
WORKDIR /app

COPY . . 
RUN go build -o result-builder .

# Start a new stage for the minimal runtime container
FROM alpine:latest

RUN apk update && apk add --no-cache ca-certificates libc6-compat

# Set the working directory inside the minimal runtime container

# Copy the built binary from the builder container into the minimal runtime container
COPY --from=builder /app/result-builder /

# Ensure the binary is executable
RUN chmod +x /result-builder

# Run your Go application
CMD ["/result-builder"]
