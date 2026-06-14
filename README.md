# Aura Backend

Go/Gin backend for Aura.

## Local Services

```bash
docker compose up -d
```

## Migrations

Install `golang-migrate` if needed:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Apply migrations:

```bash
migrate -path migrations -database "postgres://aura_user:aura_password@localhost:5433/aura_db?sslmode=disable" up
```

Check version:

```bash
migrate -path migrations -database "postgres://aura_user:aura_password@localhost:5433/aura_db?sslmode=disable" version
```

Rollback one migration:

```bash
migrate -path migrations -database "postgres://aura_user:aura_password@localhost:5433/aura_db?sslmode=disable" down 1
```

## Run

```bash
cp .env.example .env
go run ./cmd/api
go run ./cmd/worker
```

Health check:

```bash
curl http://localhost:8080/api/v1/health
```

## API Docs

Mobile integration guide:

```text
docs/API_MOBILE.md
```

Swagger/OpenAPI spec:

```text
docs/openapi.yaml
```

Open the YAML in Swagger UI, Redoc, Postman, or any OpenAPI-compatible tool.

Swagger through the API server:

```text
http://localhost:8080/swagger.html
```

Alternative static docs server:

```bash
python3 -m http.server 9000 --directory docs
```

Then open:

```text
http://localhost:9000/swagger.html
```
