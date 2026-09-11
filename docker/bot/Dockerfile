FROM golang:1.24-bookworm

# Install ffmpeg, curl and pip for yt-dlp
RUN apt-get update && apt-get install -y \
    ffmpeg \
    curl \
    git \
    python3-pip \
    python3-venv \
    libcurl4-openssl-dev \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Install latest yt-dlp via pip with impersonation support (curl-cffi)
# curl-cffi provides browser impersonation for sites like TikTok
RUN pip3 install --break-system-packages --no-cache-dir -U \
    "yt-dlp[default,curl-cffi] @ git+https://github.com/yt-dlp/yt-dlp.git"

# Verify curl-cffi is installed correctly
RUN python3 -c "from curl_cffi import requests; print('curl-cffi installed successfully')"

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

# Create shared directory for downloads
RUN mkdir -p /downloads

EXPOSE 8080

CMD ["./mediaharvester-bot"]
