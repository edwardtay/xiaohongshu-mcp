# XHS Posting Service — Design Doc

**Date:** 2026-02-23
**Approach:** Fork & extend [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp)

## Overview

Deploy an XHS (Xiaohongshu) posting service on CapRover by forking the existing `xiaohongshu-mcp` Go project. The upstream project already provides 90% of what we need — headless Chrome browser automation via go-rod, cookie-based session persistence, QR code login, image+text publishing, search, commenting, and feed browsing. We add missing HTTP endpoints for like/favorite, API key authentication, and CapRover deployment config.

## Architecture

```
CapRover (self-hosted)
└── xhs-posting-service (Docker container)
    ├── Go HTTP API (Gin framework, port 18060)
    ├── go-rod headless Chrome (google-chrome-stable)
    ├── Cookie persistence (/app/data/cookies.json)
    └── Image storage (/app/images/)

n8n (separate service) ──HTTP+API-Key──▶ xhs-posting-service
```

## Existing Endpoints (upstream, no changes needed)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /health | Health check |
| GET | /api/v1/login/status | Check if logged in |
| GET | /api/v1/login/qrcode | Get QR code for login |
| DELETE | /api/v1/login/cookies | Reset login state |
| POST | /api/v1/publish | Publish image+text note |
| POST | /api/v1/publish_video | Publish video note |
| GET | /api/v1/feeds/list | List homepage feeds |
| GET/POST | /api/v1/feeds/search | Search feeds by keyword |
| POST | /api/v1/feeds/detail | Get feed detail + comments |
| POST | /api/v1/feeds/comment | Post comment on feed |
| POST | /api/v1/feeds/comment/reply | Reply to a comment |
| POST | /api/v1/user/profile | Get user profile |
| GET | /api/v1/user/me | Get own profile |

## Changes to Add

### 1. Like/Favorite HTTP Endpoints

The service layer already has `LikeFeed`, `UnlikeFeed`, `FavoriteFeed`, `UnfavoriteFeed` methods. Wire them to HTTP routes:

```
POST   /api/v1/feeds/like       { feed_id, xsec_token }  → Like
DELETE /api/v1/feeds/like       { feed_id, xsec_token }  → Unlike
POST   /api/v1/feeds/favorite   { feed_id, xsec_token }  → Favorite
DELETE /api/v1/feeds/favorite   { feed_id, xsec_token }  → Unfavorite
```

### 2. API Key Auth Middleware

- Read `API_KEY` from environment variable
- Check `X-API-Key` header on all requests except `/health`
- Return 401 if missing/invalid

### 3. CapRover Deployment

- `captain-definition` file with Dockerfile path
- Environment variables: `ROD_BROWSER_BIN`, `COOKIES_PATH`, `API_KEY`
- Persistent volume for `/app/data` (cookies) and `/app/images`

### 4. n8n Workflow Template

JSON workflow export covering:
- Cron trigger (randomized daily posting)
- DeepSeek API HTTP request node (content generation)
- HTTP request node → POST /api/v1/publish
- Delay node (random 30min-2hr)
- HTTP request node → engagement (like/comment on search results)

## What We're NOT Changing

- Core go-rod browser automation
- Cookie/session management
- Publishing flow (title validation, image upload, tag input, schedule)
- MCP server (kept intact, harmless)
- Dockerfile (works as-is)

## Tech Stack

- **Language:** Go 1.24
- **Framework:** Gin (HTTP), go-rod (browser automation)
- **Browser:** Google Chrome (headless, in Docker)
- **Deployment:** Docker on CapRover
- **Orchestration:** n8n (separate service)
- **Content LLM:** DeepSeek API (called from n8n, not from this service)
