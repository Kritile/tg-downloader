# MediaHarvester Bot

A Telegram and MAX bot for downloading videos from YouTube and TikTok with per-user access control and an admin panel.

## Features

- 📥 Download videos from YouTube and TikTok
- 👥 Per-user access control
- 📊 Daily and monthly download limits
- 🔐 Admin panel for user management
- ⚡ Redis-based job queue
- 🐳 Fully containerized with Docker

## Tech Stack

- **Backend**: Go 1.22+
- **Database**: PostgreSQL 15
- **Queue**: Redis 7
- **Admin Panel**: Gin + HTMX + TailwindCSS
- **Telegram Bot**: go-telegram-bot-api through Telegram Local Bot API
- **MAX Bot**: max-messenger/max-bot-api-client-go/v2 through the direct MAX API (never through Xray)
- **Video Downloader**: yt-dlp

## Quick Start

### 1. Clone the repository

```bash
git clone <repository-url>
cd tg-downloader
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

Edit `.env` and set:

```env
BOT_TOKEN=your_telegram_bot_token
MAX_BOT_TOKEN=your_max_bot_token
TELEGRAM_API_ID=your_telegram_api_id
TELEGRAM_API_HASH=your_telegram_api_hash
TELEGRAM_API_URL=http://telegram-bot-api:8081
GLOBAL_DEFAULT_DAILY_LIMIT=10
GLOBAL_DEFAULT_MONTHLY_LIMIT=100
WORKER_COUNT=3
ADMIN_PORT=8080
SESSION_SECRET=your_secret_key_change_in_production
```

### 3. Start with Docker Compose

```bash
docker compose up --build
```

### 4. Access the admin panel

Open http://localhost:8080

**Default credentials:**
- Username: `admin`
- Password: `admin123`

⚠️ **Change the password immediately after first login!**

## Project Structure

```
tg-downloader/
├── bot/                    # Telegram bot service
│   ├── cmd/
│   │   └── bot/           # Bot entry point
│   └── internal/
│       ├── domain/        # Domain interfaces and errors
│       ├── repository/    # PostgreSQL repositories
│       ├── transport/     # Telegram bot handlers
│       ├── usecase/       # Business logic
│       └── worker/        # Redis queue workers
├── admin/                  # Admin panel service
│   ├── cmd/
│   │   └── admin/         # Admin entry point
│   ├── internal/
│   │   ├── domain/        # Domain interfaces
│   │   ├── repository/    # PostgreSQL repositories
│   │   ├── transport/     # HTTP handlers
│   │   └── usecase/       # Business logic
│   └── templates/         # HTML templates
├── shared/                 # Shared code
│   ├── config/            # Configuration
│   └── models/            # Domain models
├── migrations/             # Database migrations
└── docker/                 # Dockerfiles
```

## Architecture

```
┌─────────────┐     ┌──────────────┐
│   Telegram  │────▶│  Bot Service │
│   Users     │     │              │
└─────────────┘     └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Redis Queue │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Worker Pool  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   yt-dlp     │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Telegram   │
                    │   (send video)│
                    └──────────────┘

┌─────────────┐     ┌──────────────┐
│   Admin     │────▶│ Admin Panel  │
│   Browser   │     │   (Gin)      │
└─────────────┘     └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  PostgreSQL  │
                    └──────────────┘
```

## API Endpoints

### Bot

The bot can interact via Telegram and MAX:
- Send `/start` to begin
- Send a YouTube or TikTok URL to download

Telegram requests use `TELEGRAM_API_URL` through the configured SOCKS5 Xray transport. The Compose deployment shares `/downloads` between the worker and Local Bot API container, allowing Telegram to consume local file paths. MAX requests use the MAX client directly without Xray. All yt-dlp downloads use Xray.

### Admin Panel

- `GET /` - Login page
- `POST /login` - Authenticate
- `POST /logout` - Logout
- `GET /dashboard` - Overview
- `GET /users` - User management
- `GET /settings` - Global settings
- `GET /stats` - Statistics

## Database Schema

### users
- `id` - Primary key
- `telegram_id` - Unique Telegram user ID
- `username` - Telegram username
- `can_youtube` - YouTube permission (nullable)
- `can_tiktok` - TikTok permission (nullable)
- `daily_limit` - Personal daily limit (nullable)
- `monthly_limit` - Personal monthly limit (nullable)

### downloads
- `id` - Primary key
- `user_id` - Foreign key to users
- `source` - Video source (youtube/tiktok)
- `video_url` - Original URL
- `file_size` - File size in bytes
- `status` - Download status
- `created_at` - Timestamp

### settings
- `id` - Primary key
- `default_daily_limit` - Global daily limit
- `default_monthly_limit` - Global monthly limit
- `default_youtube_allowed` - YouTube enabled
- `default_tiktok_allowed` - TikTok enabled

### admins
- `id` - Primary key
- `username` - Admin username
- `password_hash` - Bcrypt hashed password

## Limit Logic

Priority order:
1. If `user.daily_limit` is set → use it
2. Else → use `settings.default_daily_limit`

Same for monthly limits.

Counters are calculated from the `downloads` table:
- Daily: `WHERE created_at >= today 00:00`
- Monthly: `WHERE created_at >= first day of month`

## Security

- URL validation (only YouTube and TikTok)
- Session-based admin authentication
- Bcrypt password hashing
- CSRF protection via sessions
- File size limit (50MB max for Telegram)

## Development

### Run tests

```bash
go test ./...
```

### Run bot locally

```bash
cd bot/cmd/bot
go run .
```

### Run admin locally

```bash
cd admin/cmd/admin
go run .
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BOT_TOKEN` | Telegram bot token | Required |
| `MAX_BOT_TOKEN` | MAX bot token | Optional |
| `TELEGRAM_API_ID` | Telegram application ID for Local Bot API | Required for local mode |
| `TELEGRAM_API_HASH` | Telegram application hash for Local Bot API | Required for local mode |
| `TELEGRAM_API_URL` | Telegram Local Bot API base URL | `http://telegram-bot-api:8081` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_URL` | Redis connection string | Required |
| `GLOBAL_DEFAULT_DAILY_LIMIT` | Default daily downloads | 10 |
| `GLOBAL_DEFAULT_MONTHLY_LIMIT` | Default monthly downloads | 100 |
| `WORKER_COUNT` | Number of download workers | 3 |
| `ADMIN_PORT` | Admin panel port | 8080 |
| `SESSION_SECRET` | Session encryption key | Required |

## License

MIT
