# Threads Affiliate Automation Platform

AI-Powered affiliate marketing automation via Threads (Meta). Supports Shopee & TikTok Shop affiliate links with organic AI content generation.

## Overview

Platform web app yang mengotomasi affiliate marketing melalui Threads:
- **Input:** User paste affiliate links dari Shopee/TikTok (5 menit/hari)
- **AI:** Generate konten organik yang tidak terlihat jualan
- **Publish:** Auto-post ke Threads via official API
- **Track:** Click tracking via custom URL shortener
- **Learn:** AI self-learning dari engagement data

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend API | Go (Fiber) |
| Frontend | React 18 + Vite + TailwindCSS |
| AI Service | Python + FastAPI + Claude API |
| Job Queue | Asynq (Go-native, Redis-backed) |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| URL Shortener | Go + Redis + Cloudflare CDN |
| Auth | JWT + Meta OAuth 2.0 |
| Deploy | Docker Compose + Nginx |

## Project Structure

```
threads-affiliate-automation/
├── cmd/
│   ├── api/            # Main API server
│   ├── worker/         # Asynq worker (background jobs)
│   └── shortener/      # URL shortener service
├── internal/
│   ├── auth/           # JWT + OAuth logic
│   ├── config/         # App configuration
│   ├── database/       # DB connection & migrations
│   ├── handler/        # HTTP handlers (Fiber)
│   ├── middleware/     # Auth, rate limit, CORS
│   ├── model/          # Data models
│   ├── repository/     # Database queries
│   ├── service/        # Business logic
│   ├── queue/          # Asynq job definitions
│   ├── shortener/      # URL shortener logic
│   ├── threads/        # Threads API client
│   └── ai/             # AI service client
├── pkg/
│   ├── validator/      # Input validation
│   └── utils/          # Shared utilities
├── web/frontend/       # React frontend
├── migrations/         # SQL migrations
├── scripts/            # Utility scripts
├── deployments/        # Docker & deployment configs
├── docs/plans/         # Implementation plans
└── tests/              # Integration tests
```

## Quick Start

```bash
# Prerequisites
# - Go 1.22+
# - PostgreSQL 16
# - Redis 7
# - Node.js 20+ (for frontend)

# Setup
cp .env.example .env
# Edit .env with your credentials

# Run migrations
go run cmd/api/main.go migrate

# Start API server
go run cmd/api/main.go

# Start worker
go run cmd/worker/main.go

# Start URL shortener
go run cmd/shortener/main.go

# Frontend
cd web/frontend && npm install && npm run dev
```

## Documentation

- [PRD v2.0](/docs/prd-v2.0.pdf)
- [Workflow v2.0](/docs/workflow-v2.0.pdf)
- [Phase 0: Validation](/docs/plans/phase0-validation.md)

## License

MIT
