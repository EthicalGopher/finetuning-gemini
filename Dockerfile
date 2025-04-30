FROM golang:1.24-alpine as builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o finetune

# Use a minimal alpine image for the final stage
FROM alpine:3.19

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/finetune .

# Expose the port the app runs on
EXPOSE 3000

# Command to run the executable
CMD ["./finetune"]
