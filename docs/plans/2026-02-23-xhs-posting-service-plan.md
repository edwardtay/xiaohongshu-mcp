# XHS Posting Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fork xiaohongshu-mcp, add like/favorite HTTP endpoints, API key auth, CapRover config, and n8n workflow template.

**Architecture:** Extend the existing Go HTTP API server (Gin + go-rod) with 4 new engagement endpoints, a simple API key middleware, and deployment config for CapRover. The n8n workflow template demonstrates the full content-generation-to-posting pipeline using DeepSeek.

**Tech Stack:** Go 1.24, Gin, go-rod, Docker, CapRover, n8n, DeepSeek API

---

### Task 1: Fork and Clone the Upstream Repo

**Files:**
- Create: `/home/edwardtay/2-projects/xiaohongshu/` (populated via git clone)

**Step 1: Fork on GitHub**

Run:
```bash
gh repo fork xpzouying/xiaohongshu-mcp --clone=false
```
Expected: Fork created under your GitHub account.

**Step 2: Clone your fork into the project directory**

Run:
```bash
cd /home/edwardtay/2-projects
rm -rf xiaohongshu
gh repo clone edwardtay/xiaohongshu-mcp xiaohongshu
cd xiaohongshu
```
Expected: Full repo cloned with all upstream code.

**Step 3: Verify the build compiles**

Run:
```bash
cd /home/edwardtay/2-projects/xiaohongshu
go build -o /dev/null .
```
Expected: Clean build, no errors.

**Step 4: Commit marker (no code change, just verify)**

No commit needed — repo is already initialized from upstream.

---

### Task 2: Add Like/Favorite HTTP Endpoints

The service layer methods `LikeFeed`, `UnlikeFeed`, `FavoriteFeed`, `UnfavoriteFeed` already exist in `service.go`. We need to add HTTP handler functions and wire them to routes.

**Files:**
- Modify: `handlers_api.go` (add 4 handler functions)
- Modify: `routes.go` (add 4 routes)
- Modify: `types.go` (add request type)

**Step 1: Add the request type to `types.go`**

Add after the existing `ReplyCommentRequest` struct:

```go
// LikeFavoriteRequest 点赞/收藏请求
type LikeFavoriteRequest struct {
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
}
```

**Step 2: Add 4 handler functions to `handlers_api.go`**

Add at the end of the file, before the closing:

```go
// likeFeedHandler 点赞笔记
func (s *AppServer) likeFeedHandler(c *gin.Context) {
	var req LikeFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	result, err := s.xiaohongshuService.LikeFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIKE_FAILED",
			"点赞失败", err.Error())
		return
	}

	respondSuccess(c, result, result.Message)
}

// unlikeFeedHandler 取消点赞
func (s *AppServer) unlikeFeedHandler(c *gin.Context) {
	var req LikeFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	result, err := s.xiaohongshuService.UnlikeFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "UNLIKE_FAILED",
			"取消点赞失败", err.Error())
		return
	}

	respondSuccess(c, result, result.Message)
}

// favoriteFeedHandler 收藏笔记
func (s *AppServer) favoriteFeedHandler(c *gin.Context) {
	var req LikeFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	result, err := s.xiaohongshuService.FavoriteFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "FAVORITE_FAILED",
			"收藏失败", err.Error())
		return
	}

	respondSuccess(c, result, result.Message)
}

// unfavoriteFeedHandler 取消收藏
func (s *AppServer) unfavoriteFeedHandler(c *gin.Context) {
	var req LikeFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	result, err := s.xiaohongshuService.UnfavoriteFeed(c.Request.Context(), req.FeedID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "UNFAVORITE_FAILED",
			"取消收藏失败", err.Error())
		return
	}

	respondSuccess(c, result, result.Message)
}
```

**Step 3: Wire routes in `routes.go`**

Inside the `api := router.Group("/api/v1")` block, add after the existing `api.POST("/feeds/comment/reply", ...)` line:

```go
		api.POST("/feeds/like", appServer.likeFeedHandler)
		api.DELETE("/feeds/like", appServer.unlikeFeedHandler)
		api.POST("/feeds/favorite", appServer.favoriteFeedHandler)
		api.DELETE("/feeds/favorite", appServer.unfavoriteFeedHandler)
```

**Step 4: Verify build**

Run:
```bash
cd /home/edwardtay/2-projects/xiaohongshu
go build -o /dev/null .
```
Expected: Clean build.

**Step 5: Commit**

```bash
git add types.go handlers_api.go routes.go
git commit -m "feat: add like/favorite HTTP endpoints"
```

---

### Task 3: Add API Key Auth Middleware

**Files:**
- Modify: `middleware.go` (add auth middleware function)
- Modify: `routes.go` (apply middleware to `/api/v1` group)

**Step 1: Add auth middleware to `middleware.go`**

Add at the end of the file:

```go
// apiKeyAuthMiddleware API Key 认证中间件
func apiKeyAuthMiddleware() gin.HandlerFunc {
	apiKey := os.Getenv("API_KEY")

	return func(c *gin.Context) {
		// 如果没有配置 API_KEY，跳过认证
		if apiKey == "" {
			c.Next()
			return
		}

		key := c.GetHeader("X-API-Key")
		if key == "" {
			respondError(c, http.StatusUnauthorized, "MISSING_API_KEY",
				"缺少 API Key", "请在请求头中设置 X-API-Key")
			c.Abort()
			return
		}

		if key != apiKey {
			respondError(c, http.StatusUnauthorized, "INVALID_API_KEY",
				"API Key 无效", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
```

Add `"os"` to the imports at the top of `middleware.go`.

**Step 2: Apply middleware to the API group in `routes.go`**

Change the API group definition from:

```go
	api := router.Group("/api/v1")
```

To:

```go
	api := router.Group("/api/v1")
	api.Use(apiKeyAuthMiddleware())
```

This keeps `/health` and `/mcp` unauthenticated while protecting all `/api/v1/*` routes.

**Step 3: Verify build**

Run:
```bash
cd /home/edwardtay/2-projects/xiaohongshu
go build -o /dev/null .
```
Expected: Clean build.

**Step 4: Commit**

```bash
git add middleware.go routes.go
git commit -m "feat: add API key auth middleware for /api/v1 routes"
```

---

### Task 4: Add CapRover Deployment Config

**Files:**
- Create: `captain-definition`
- Create: `deploy/caprover/README.md`

**Step 1: Create `captain-definition`**

```json
{
  "schemaVersion": 2,
  "dockerfilePath": "./Dockerfile"
}
```

**Step 2: Create `deploy/caprover/README.md`**

```markdown
# CapRover Deployment

## Setup

1. Create a new app in CapRover (e.g., `xhs-service`)
2. Enable **persistent storage**:
   - `/app/data` → for cookies.json
   - `/app/images` → for uploaded images
3. Set environment variables:
   - `ROD_BROWSER_BIN=/usr/bin/google-chrome`
   - `COOKIES_PATH=/app/data/cookies.json`
   - `API_KEY=<your-secret-key>`
4. Deploy via CLI:
   ```bash
   caprover deploy -a xhs-service
   ```

## First Login

After deployment, get the QR code to log in:

```bash
curl https://xhs-service.your-domain.com/api/v1/login/qrcode \
  -H "X-API-Key: your-key"
```

This returns a base64 QR code image. Decode it and scan with the XHS app.

## Verify Login

```bash
curl https://xhs-service.your-domain.com/api/v1/login/status \
  -H "X-API-Key: your-key"
```

## Test Publishing

```bash
curl -X POST https://xhs-service.your-domain.com/api/v1/publish \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试笔记",
    "content": "这是一条测试内容",
    "images": ["https://picsum.photos/800/600"],
    "tags": ["测试"]
  }'
```
```

**Step 3: Commit**

```bash
git add captain-definition deploy/caprover/README.md
git commit -m "feat: add CapRover deployment config"
```

---

### Task 5: Create n8n Workflow Template

**Files:**
- Create: `examples/n8n/xhs-auto-post-workflow.json`

**Step 1: Write the n8n workflow JSON**

This workflow does:
1. **Schedule Trigger** — fires daily at a random hour (configured as 10am, user adjusts)
2. **DeepSeek Generate** — calls DeepSeek API to generate a XHS post (title, content, tags)
3. **Parse Response** — extracts structured fields from DeepSeek response
4. **Publish to XHS** — calls the posting service
5. **Wait** — random delay 1-3 hours
6. **Search Trending** — find trending feeds in the niche
7. **Like & Comment** — engage with top results

```json
{
  "name": "XHS Auto Post + Engage",
  "nodes": [
    {
      "parameters": {
        "rule": {
          "interval": [{ "field": "hours", "hoursInterval": 24 }]
        }
      },
      "name": "Daily Trigger",
      "type": "n8n-nodes-base.scheduleTrigger",
      "typeVersion": 1.2,
      "position": [240, 300]
    },
    {
      "parameters": {
        "url": "https://api.deepseek.com/chat/completions",
        "authentication": "genericCredentialType",
        "genericAuthType": "httpHeaderAuth",
        "sendBody": true,
        "specifyBody": "json",
        "jsonBody": "={\n  \"model\": \"deepseek-chat\",\n  \"messages\": [\n    {\n      \"role\": \"system\",\n      \"content\": \"你是一个小红书内容创作者。根据给定的主题，生成一篇小红书笔记。返回JSON格式：{\\\"title\\\": \\\"标题(最多20字)\\\", \\\"content\\\": \\\"正文(200-500字)\\\", \\\"tags\\\": [\\\"标签1\\\", \\\"标签2\\\", \\\"标签3\\\"]}\"\n    },\n    {\n      \"role\": \"user\",\n      \"content\": \"主题：生活方式\"\n    }\n  ],\n  \"response_format\": { \"type\": \"json_object\" }\n}",
        "options": {}
      },
      "name": "DeepSeek Generate",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.2,
      "position": [460, 300],
      "credentials": {
        "httpHeaderAuth": {
          "name": "DeepSeek API Key"
        }
      }
    },
    {
      "parameters": {
        "jsCode": "const response = JSON.parse($input.first().json.choices[0].message.content);\nreturn [{ json: response }];"
      },
      "name": "Parse Response",
      "type": "n8n-nodes-base.code",
      "typeVersion": 2,
      "position": [680, 300]
    },
    {
      "parameters": {
        "url": "={{$env.XHS_SERVICE_URL}}/api/v1/publish",
        "sendHeaders": true,
        "headerParameters": {
          "parameters": [
            { "name": "X-API-Key", "value": "={{$env.XHS_API_KEY}}" }
          ]
        },
        "sendBody": true,
        "specifyBody": "json",
        "jsonBody": "={\n  \"title\": \"{{ $json.title }}\",\n  \"content\": \"{{ $json.content }}\",\n  \"images\": [\"https://picsum.photos/800/600\"],\n  \"tags\": {{ JSON.stringify($json.tags) }}\n}",
        "options": {}
      },
      "name": "Publish to XHS",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.2,
      "position": [900, 300]
    },
    {
      "parameters": {
        "amount": "={{Math.floor(Math.random() * 120) + 60}}",
        "unit": "minutes"
      },
      "name": "Random Delay",
      "type": "n8n-nodes-base.wait",
      "typeVersion": 1.1,
      "position": [1120, 300]
    },
    {
      "parameters": {
        "url": "={{$env.XHS_SERVICE_URL}}/api/v1/feeds/search",
        "sendHeaders": true,
        "headerParameters": {
          "parameters": [
            { "name": "X-API-Key", "value": "={{$env.XHS_API_KEY}}" }
          ]
        },
        "sendBody": true,
        "specifyBody": "json",
        "jsonBody": "{ \"keyword\": \"生活方式\" }",
        "options": {}
      },
      "name": "Search Trending",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.2,
      "position": [1340, 300]
    },
    {
      "parameters": {
        "jsCode": "const feeds = $input.first().json.data?.feeds || [];\nconst top3 = feeds.slice(0, 3);\nreturn top3.map(f => ({\n  json: {\n    feed_id: f.id || f.note_id,\n    xsec_token: f.xsec_token || ''\n  }\n}));"
      },
      "name": "Pick Top 3 Feeds",
      "type": "n8n-nodes-base.code",
      "typeVersion": 2,
      "position": [1560, 300]
    },
    {
      "parameters": {
        "url": "={{$env.XHS_SERVICE_URL}}/api/v1/feeds/like",
        "sendHeaders": true,
        "headerParameters": {
          "parameters": [
            { "name": "X-API-Key", "value": "={{$env.XHS_API_KEY}}" }
          ]
        },
        "sendBody": true,
        "specifyBody": "json",
        "jsonBody": "={\n  \"feed_id\": \"{{ $json.feed_id }}\",\n  \"xsec_token\": \"{{ $json.xsec_token }}\"\n}",
        "options": {}
      },
      "name": "Like Feed",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.2,
      "position": [1780, 300]
    }
  ],
  "connections": {
    "Daily Trigger": { "main": [[{ "node": "DeepSeek Generate", "type": "main", "index": 0 }]] },
    "DeepSeek Generate": { "main": [[{ "node": "Parse Response", "type": "main", "index": 0 }]] },
    "Parse Response": { "main": [[{ "node": "Publish to XHS", "type": "main", "index": 0 }]] },
    "Publish to XHS": { "main": [[{ "node": "Random Delay", "type": "main", "index": 0 }]] },
    "Random Delay": { "main": [[{ "node": "Search Trending", "type": "main", "index": 0 }]] },
    "Search Trending": { "main": [[{ "node": "Pick Top 3 Feeds", "type": "main", "index": 0 }]] },
    "Pick Top 3 Feeds": { "main": [[{ "node": "Like Feed", "type": "main", "index": 0 }]] }
  },
  "settings": {
    "executionOrder": "v1"
  }
}
```

Note to implementer: The `images` field in "Publish to XHS" uses a placeholder URL. In production, replace with actual image URLs or local paths. The DeepSeek prompt topic ("生活方式") should be customized per niche.

**Step 2: Commit**

```bash
git add examples/n8n/xhs-auto-post-workflow.json
git commit -m "feat: add n8n auto-post workflow template with DeepSeek"
```

---

### Task 6: Final Verification and Push

**Step 1: Full build check**

Run:
```bash
cd /home/edwardtay/2-projects/xiaohongshu
go build -o /dev/null .
```
Expected: Clean build.

**Step 2: Run existing tests**

Run:
```bash
cd /home/edwardtay/2-projects/xiaohongshu
go test ./... 2>&1 | head -30
```
Expected: All existing tests pass (or skip if they require a browser).

**Step 3: Docker build test**

Run:
```bash
cd /home/edwardtay/2-projects/xiaohongshu
docker build -t xhs-service:local .
```
Expected: Image builds successfully. This will take a few minutes (downloads Chrome).

**Step 4: Smoke test with Docker**

Run:
```bash
docker run --rm -d --name xhs-test -p 18060:18060 -e API_KEY=test123 xhs-service:local
sleep 5
curl -s http://localhost:18060/health | python3 -m json.tool
curl -s -H "X-API-Key: test123" http://localhost:18060/api/v1/login/status | python3 -m json.tool
curl -s http://localhost:18060/api/v1/login/status | python3 -m json.tool  # should return 401
docker stop xhs-test
```
Expected:
- `/health` returns `{"success": true, "data": {"status": "healthy", ...}}`
- `/api/v1/login/status` with key returns login status
- `/api/v1/login/status` without key returns 401 unauthorized

**Step 5: Push to remote**

Run:
```bash
git push origin main
```
