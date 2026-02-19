
## Project Name

**MediaHarvester Bot**

## Project Type

Public Telegram Bot for downloading videos from YouTube and TikTok
With admin panel and per-user access control

---

# 1️⃣ GLOBAL RULES (MANDATORY)

1. All development MUST be done inside Docker.
2. No host-level commands allowed.
3. Every meaningful change MUST:

   * Be committed
   * Include clear commit message
4. Every business rule MUST have unit tests.
5. All services MUST be containerized.
6. Use PostgreSQL and Redis.
7. Follow Clean Architecture.
8. No hardcoded secrets.
9. Use environment variables only.
10. The bot must delete downloaded files immediately after sending.

---

# 2️⃣ TECHNOLOGY STACK (STRICT)

## Backend (Bot)

* Language: Go 1.22+
* Framework: standard library or minimal framework
* Telegram: go-telegram-bot-api
* Downloader: yt-dlp (via CLI execution)
* Queue: Redis + worker pool
* Logging: zap (structured logging)

## Admin Panel

* Backend: Go (Gin)
* Frontend: HTMX + TailwindCSS (server-rendered)
* Auth: session-based
* Password hashing: bcrypt

## Database

* PostgreSQL 15+
* Migrations: golang-migrate
* ORM: GORM or sqlx

## Queue

* Redis 7+
* Use Redis list or stream
* Background worker processes

## Infrastructure

* Docker
* Docker Compose
* No Kubernetes
* No external cloud dependencies

---

# 3️⃣ SYSTEM ARCHITECTURE

The system must contain:

```
/bot
/admin
/shared
/migrations
/docker
```

Services in docker-compose:

* bot
* admin
* postgres
* redis
* yt-dlp container or installed in bot image

---

# 4️⃣ FUNCTIONAL REQUIREMENTS

## 4.1 Bot Behavior

When a user sends a message:

1. Validate URL
2. Detect source:

   * youtube
   * tiktok
3. Check:

   * If user exists → else create
   * Permission for source
   * Daily limit
   * Monthly limit
4. If allowed:

   * Push task to Redis queue
   * Notify user "Downloading..."
5. Worker:

   * Pull job
   * Execute yt-dlp
   * Send video as Telegram video message
   * Delete file immediately
   * Log download
   * Update counters
6. If limit exceeded:

   * Send limit message
7. If not allowed:

   * Send permission denied

---

# 5️⃣ LIMIT LOGIC (STRICT)

## Priority:

1. If user.daily_limit != NULL → use it
2. Else → use global_daily_limit

Same for monthly.

Counters must be calculated by querying downloads table:

* daily → WHERE created_at >= today 00:00
* monthly → WHERE created_at >= first day of month

No in-memory counters allowed.

---

# 6️⃣ DATABASE SCHEMA

## users

* id (PK)
* telegram_id (unique)
* username
* can_youtube (bool, nullable)
* can_tiktok (bool, nullable)
* daily_limit (int, nullable)
* monthly_limit (int, nullable)
* created_at
* updated_at

---

## downloads

* id
* user_id (FK)
* source (varchar)
* video_url
* created_at

---

## settings

* id
* default_daily_limit
* default_monthly_limit
* default_youtube_allowed
* default_tiktok_allowed

---

## admins

* id
* username
* password_hash
* created_at

---

# 7️⃣ ADMIN PANEL REQUIREMENTS

## Must include:

### User Management

* Search by telegram_id
* Toggle:

  * YouTube access
  * TikTok access
* Set personal limits
* Reset limits

### Global Settings

* Set default daily limit
* Set default monthly limit
* Enable/disable YouTube globally
* Enable/disable TikTok globally

### Statistics Page

* Downloads today
* Downloads this month
* Top 10 users

---

# 8️⃣ REDIS QUEUE REQUIREMENTS

* Use Redis for job queue
* Jobs must include:

  * user_id
  * url
  * source
* Workers must:

  * Retry 2 times
  * Log failures
* Configurable worker count via ENV

---

# 9️⃣ FILE HANDLING RULES

* Download path: /tmp/downloads
* After successful Telegram send:

  * Immediately delete file
* If send fails:

  * Delete file anyway
* No persistent video storage allowed

---

# 🔟 TELEGRAM LIMITS

Bot API max file size: 50MB
If file > 50MB:

* Abort
* Send message: "File too large"

No self-hosted Bot API allowed.

---

# 11️⃣ SECURITY REQUIREMENTS

* Validate URL format
* Accept only:

  * youtube.com
  * youtu.be
  * tiktok.com
* Rate limit per user (anti-spam middleware)
* Protect admin routes with authentication
* CSRF protection enabled

---

# 12️⃣ ENVIRONMENT VARIABLES

```
BOT_TOKEN=
DATABASE_URL=
REDIS_URL=
GLOBAL_DEFAULT_DAILY_LIMIT=
GLOBAL_DEFAULT_MONTHLY_LIMIT=
WORKER_COUNT=3
ADMIN_PORT=8080
SESSION_SECRET=
```

---

# 13️⃣ DOCKER REQUIREMENTS

All services must be defined in docker-compose.yml.

Must include:

* postgres with volume
* redis with volume
* bot service
* admin service

Bot image must include:

* yt-dlp installed
* ffmpeg installed

No manual setup allowed.

Project must start with:

```
docker compose up --build
```

---

# 14️⃣ TESTING REQUIREMENTS (MANDATORY)

AI must implement:

## Unit Tests

* Permission resolution logic
* Daily limit logic
* Monthly limit logic
* Source detection
* URL validation

## Integration Tests

* Redis queue processing
* Database interaction
* Worker job lifecycle

Tests must:

* Run inside Docker
* Use separate test database
* Have >70% coverage for business logic

---

# 15️⃣ GIT DISCIPLINE (MANDATORY)

AI must:

* Commit after each logical milestone
* Use conventional commit messages:

Examples:

```
feat(bot): implement youtube detection
feat(queue): add redis worker
feat(admin): add user management page
test(limits): add daily limit tests
refactor(domain): extract permission service
```

No giant commits allowed.

---

# 16️⃣ CLEAN ARCHITECTURE STRUCTURE

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

Dependencies must point inward only.

No circular imports.

---

# 17️⃣ FUTURE EXTENSION PREPARATION

Code must be designed to support future:

* Paid subscriptions
* Role-based access (free/premium)
* Payment integration
* S3 storage
* Instagram support

Architecture must allow extension without rewriting core logic.

---

# 18️⃣ NON-FUNCTIONAL REQUIREMENTS

* Max 100 downloads/hour
* Graceful shutdown
* Context-based cancellation
* Structured logging
* Healthcheck endpoint for bot and admin

---

# 19️⃣ ACCEPTANCE CRITERIA

The system is complete when:

* User can send YouTube link and receive video
* User can send TikTok link and receive video
* Limits enforced
* Admin panel can modify user permissions
* Redis queue processes jobs
* Files deleted after sending
* All services run in Docker
* Tests pass

---

If needed, I can also provide:

* Redis job schema definition
* Full folder tree example
* Database migration files
* Initial commit breakdown plan
* ER diagram
* CI pipeline spec

Tell me if you want CI/CD included.

