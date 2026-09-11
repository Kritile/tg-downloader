## Project Name

**MediaHarvester Bot**

## Project Type

Public Telegram Bot for downloading videos from YouTube and TikTok
With admin panel, per-user access control, Redis queue, containerized SOCKS5 VPN for YouTube traffic, and video format selection

---

# 1️⃣ GLOBAL RULES

* All development **inside Docker**.
* No host-level commands allowed.
* Every meaningful change MUST:

  * Be committed with a clear message
* Every business rule MUST have unit tests.
* Use **PostgreSQL** and **Redis**.
* Follow **Clean Architecture**.
* No hardcoded secrets — use env variables.
* Bot must **delete downloaded files immediately after sending**.
* Telegram Local Bot API outbound traffic MUST go through the isolated **AmneziaWG 3 sidecar**; downloader traffic uses the configured SOCKS5 Xray endpoint.
* MAX API traffic, including inbound updates and outbound uploads, MUST use a direct connection.

---

# 2️⃣ TECHNOLOGY STACK

* **Backend (Bot)**: Go 1.22+, Telegram go-telegram-bot-api, yt-dlp (CLI), Redis worker pool, zap logging
* **Admin Panel**: Go (Gin), HTMX + TailwindCSS, session auth, bcrypt passwords
* **Database**: PostgreSQL 15+, golang-migrate, GORM/sqlx
* **Queue**: Redis 7+, list or stream
* **VPN**: AmneziaWG 3 sidecar for Telegram and configured SOCKS5 endpoint for downloads
* **Downloader**: yt-dlp with `--list-formats` for format selection

---

# 3️⃣ SYSTEM ARCHITECTURE

```
docker-compose.yml:

services:
  bot:
    build: ./bot
    depends_on: [postgres, redis, telegram-bot-api]
    environment:
      - BOT_TOKEN
      - DATABASE_URL
      - REDIS_URL
      - WORKER_COUNT
      - SESSION_SECRET
    networks:
      - internal

  admin:
    build: ./admin
    depends_on: [postgres, redis]
    environment:
      - DATABASE_URL
      - SESSION_SECRET
    networks:
      - internal

  postgres:
    image: postgres:15
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_PASSWORD=secret
    networks:
      - internal

  redis:
    image: redis:7
    volumes:
      - redisdata:/data
    networks:
      - internal

  telegram-vpn:
    build: ./docker/telegram-vpn
    cap_add: [NET_ADMIN]
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - ./docker/telegram-vpn/config/awg0.conf:/etc/amneziawg/awg0.conf:ro
    networks:
      - internal

  telegram-bot-api:
    image: aiogram/telegram-bot-api:latest
    network_mode: service:telegram-vpn

networks:
  internal:

volumes:
  pgdata:
  redisdata:
```

* The Local Bot API container shares a network namespace with the AmneziaWG 3 sidecar
* The bot routes yt-dlp requests via the independent SOCKS5 proxy
* MAX API requests bypass VPN

---

# 4️⃣ FUNCTIONAL REQUIREMENTS

## 4.1 Bot Behavior

1. Receive video URL from user
2. Detect source:

   * YouTube → route via SOCKS5 VPN
   * TikTok → normal traffic
3. Check user existence, else create
4. Check permission for source
5. Check daily/monthly limits
6. **Format selection:**

   * Query yt-dlp `--list-formats`
   * Send available formats as Telegram buttons
   * User selects format
7. Push download job to Redis queue with chosen format
8. Notify user “Downloading…”
9. Worker executes yt-dlp with chosen format
10. Send video as Telegram video message
11. Delete file immediately after sending
12. Log download, update counters
13. Handle limit exceeded or blocked user with proper message

---

# 5️⃣ LIMIT LOGIC

* Personal limit overrides global limit
* Global limit applies if personal limit null
* Daily/monthly counters computed from downloads table
* No in-memory counters

---

# 6️⃣ DATABASE SCHEMA

* **users**: telegram_id, username, can_youtube, can_tiktok, daily_limit, monthly_limit
* **downloads**: user_id, source, video_url, format, created_at
* **settings**: default_daily_limit, default_monthly_limit, default_youtube_allowed, default_tiktok_allowed
* **admins**: username, password_hash

---

# 7️⃣ ADMIN PANEL

* Manage users (permissions, limits)
* Manage global settings (daily/monthly limit, enable/disable YouTube/TikTok)
* Statistics (downloads today/month, top 10 users)

---

# 8️⃣ REDIS QUEUE

* Jobs include: user_id, url, source, chosen format
* Worker retries 2 times on failure
* Configurable worker count via ENV
* YouTube jobs → routed through SOCKS5 VPN

---

# 9️⃣ FILE HANDLING

* `/downloads` shared between bot and Telegram Local Bot API containers
* Delete after sending
* Delete on error as well

---

# 🔟 TELEGRAM LIMITS

* Max file size: 50MB
* If >50MB → abort and send message

---

# 11️⃣ SECURITY

* Validate URL domain
* Rate limit per user
* Protect admin routes
* CSRF protection enabled

---

# 12️⃣ ENV VARIABLES

```
TELEGRAM_BOT_TOKEN=
DATABASE_URL=
REDIS_URL=
GLOBAL_DEFAULT_DAILY_LIMIT=
GLOBAL_DEFAULT_MONTHLY_LIMIT=
WORKER_COUNT=3
ADMIN_PORT=8080
SESSION_SECRET=
TELEGRAM_AWG_CONFIG_PATH=./docker/telegram-vpn/config/awg0.conf
XRAY_SOCKS5_PROXY=
```

---

# 13️⃣ TESTING REQUIREMENTS

### Unit Tests

* Permission logic
* Daily/monthly limits
* Source detection
* URL validation
* Format selection logic

### Integration Tests

* Redis queue processing
* Database interaction
* Worker job lifecycle
* VPN routing check for YouTube
* Format selection → download → send workflow

Tests run inside Docker with separate test database.

---

# 14️⃣ GIT DISCIPLINE

* Commit after each milestone
* Conventional commit messages:

```
feat(bot): add youtube vpn routing
feat(bot): add format selection via --list-formats
test(limits): add daily/monthly limit tests
refactor(domain): extract permission service
```

---

# 15️⃣ CLEAN ARCHITECTURE

```
internal/
  domain/
  usecase/
  repository/
  transport/
  worker/
cmd/
  bot/
  admin/
```

* Dependencies inward only
* No circular imports

---

# 16️⃣ ACCEPTANCE CRITERIA

* Bot sends TikTok videos normally
* Bot sends YouTube videos via **SOCKS5 VPN**
* User can select video format before download
* Limits enforced
* Admin panel manages users & global settings
* Redis queue works
* Files deleted after sending
* Docker-compose runs all services cleanly
* Tests pass
