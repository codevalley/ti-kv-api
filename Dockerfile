# Use the official Go image as a base image for the build stage
FROM golang:1.21 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go mod and sum files, and download the dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code into the container
COPY . .

# Build the application
RUN go build -o main .

# Use a full Debian base for the runtime stage
ENTRYPOINT ["/app/main"]

