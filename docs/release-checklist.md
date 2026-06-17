# Release Checklist

## Preflight

```bash
git status --short
go test ./...
```

Confirm production env values:

```text
APP_ENV=production
DATABASE_URL set
REDIS_HOST and REDIS_PORT set
JWT_SECRET, ADMIN_JWT_SECRET, OTP_SECRET are unique 32+ char secrets
CORS_ALLOWED_ORIGINS is not *
OTP_DEV_BYPASS_ENABLED=false
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REDIS_REQUIRED_FOR_AUTH=true
```

## Build and Migrate

```bash
docker build -t aura-api:$(git rev-parse --short HEAD) .
pg_dump "$DATABASE_URL" > backup_before_release.sql
pg_dump "$DATABASE_URL" --format=custom --file="backups/pre_release_$(date -u +%Y%m%dT%H%M%SZ).dump"
migrate -path migrations -database "$DATABASE_URL" up
```

## Bootstrap Admin

Run only when the target database has no active `SUPER_ADMIN`.

```bash
BOOTSTRAP_ADMIN_SECRET=<32+ chars> \
go run ./cmd/bootstrap-admin --email=admin@example.com --name="Super Admin"
```

## Smoke Checks

Start the API:

```bash
go run ./cmd/api
```

Verify:

```bash
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8080/api/v1/health
curl -fsS http://localhost:8080/openapi.yaml >/dev/null
curl -fsS http://localhost:8080/swagger >/dev/null
```

Manual checks:

```text
Mobile token cannot call /api/v1/admin/*
Admin token cannot call mobile protected routes
Disabled, locked, or must-change-password admins cannot access protected admin routes
Admin capabilities return games=false, pkBattle=false, greedyGame=false
PK routes such as /api/v1/lives/test/pk/active and /api/v1/admin/pk-battles return normal 404s
POST /api/v1/auth/request-otp rate limits after configured threshold
POST /api/v1/admin/auth/login rate limits after configured threshold
Rate-limited responses include error.code=RATE_LIMITED and error.retryAfterSeconds
Payment-like mutations reject missing Idempotency-Key
Invalid UUID path params return clean 400 responses
Admin mutations create audit rows with request_id
Admin audit logs remain insert-only
No public admin signup route exists
```

## Rollback

1. Stop the new process or container.
2. Start the previous image tag with the previous env.
3. If schema rollback is required, restore the pre-release backup into a new database and point the app to it.
4. Re-run the health checks before reopening traffic.
