FROM golang:1.21.5

WORKDIR /app

# Copy the Go module files to the working directory
COPY go.mod go.sum ./

RUN go mod download

# Copy the source code to the working directory
COPY . .

# Build the Go application
RUN go build

# Set the command to run the application when the container starts
CMD ["./resolver"]
