FROM golang:1.22-bookworm

# Install ffmpeg, curl and pip for yt-dlp
RUN apt-get update && apt-get install -y \
    ffmpeg \
    curl \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/*

# Install latest yt-dlp via pip with impersonation support (curl-cffi)
RUN pip3 install --break-system-packages -U yt-dlp curl-cffi

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download || true

# Copy source code
COPY . .

# Download dependencies after copying source
RUN go mod download

# Build the bot
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mediaharvester-bot ./bot/cmd/bot

# Build the migrate tool
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate ./cmd/migrate

# Create temp directory for downloads
RUN mkdir -p /tmp/downloads

EXPOSE 8080

CMD ["./mediaharvester-bot"]
