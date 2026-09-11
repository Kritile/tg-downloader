FROM golang:1.24-bookworm

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download || true

# Copy source code
COPY . .

# Download dependencies after copying source
RUN go mod download

# Build the admin panel
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mediaharvester-admin ./admin/cmd/admin

EXPOSE 8080

CMD ["./mediaharvester-admin"]
