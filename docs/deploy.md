# Deploy

## Build

```bash
docker build -t aura-api:$(git rev-parse --short HEAD) .
```

## Environment

Create a production env file from `.env.example` and set real values.

Required production checks:

```text
APP_ENV=production
CORS_ALLOWED_ORIGINS=https://your-admin.example,https://your-app.example
JWT_SECRET=<32+ chars, not a placeholder>
ADMIN_JWT_SECRET=<32+ chars, not a placeholder>
OTP_SECRET=<32+ chars, not a placeholder>
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REDIS_REQUIRED_FOR_AUTH=true
OTP_DEV_BYPASS_ENABLED=false
```

Do not use `CORS_ALLOWED_ORIGINS=*` in production.

## Migrate

Take a pre-release backup first:

```bash
pg_dump "$DATABASE_URL" > backup_before_release.sql
```

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

## First Admin

Run once only on a fresh deployment:

```bash
BOOTSTRAP_ADMIN_SECRET=<32+ chars> \
go run ./cmd/bootstrap-admin --email=admin@example.com --name="Super Admin"
```

For Docker:

```bash
docker run --rm --env-file .env.production aura-api:$(git rev-parse --short HEAD) \
	/app/bootstrap-admin --email=admin@example.com --name="Super Admin"
```

## Run API

```bash
docker run -d --name aura-api \
	--env-file .env.production \
	-p 8080:8080 \
	aura-api:$(git rev-parse --short HEAD)
```

Health:

```bash
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8080/api/v1/health
```

Smoke checks:

```text
Swagger is reachable at /swagger
Mobile token cannot call /api/v1/admin/*
Admin token cannot call mobile protected routes
Admin capabilities keep games, pkBattle, and greedyGame false
/api/v1/lives/test/pk/active and /api/v1/admin/pk-battles return normal 404s
Payment-like mutations reject missing Idempotency-Key
Invalid UUID path params return clean 400 responses
Admin audit rows include request_id after admin mutations
```

## Rollback

1. Stop the new container.
2. Start the previous known-good image tag with the same env file.
3. Do not run destructive migrations without a fresh backup.
4. If a migration rollback is required, run it manually and verify `/health`.
