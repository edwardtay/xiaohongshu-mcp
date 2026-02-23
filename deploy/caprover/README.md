# CapRover Deployment

## Setup

1. Create a new app in CapRover (e.g., `xhs-service`)
2. Enable **persistent storage**:
   - `/app/data` -> for cookies.json
   - `/app/images` -> for uploaded images
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
