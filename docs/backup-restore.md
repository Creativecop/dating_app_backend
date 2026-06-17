# Backup and Restore

## Backup

Run a logical backup before migrations, releases, or manual data fixes.

```bash
mkdir -p backups
pg_dump "$DATABASE_URL" \
	--format=custom \
	--file="backups/aura_$(date -u +%Y%m%dT%H%M%SZ).dump"
```

For a plain SQL backup:

```bash
pg_dump "$DATABASE_URL" > "backups/aura_$(date -u +%Y%m%dT%H%M%SZ).sql"
```

Store backups outside the runtime host and restrict access. Backups can contain phone numbers, emails, payment references, audit metadata, and moderation history.

## Restore to a New Database

Custom-format backup:

```bash
createdb "$RESTORE_DATABASE_NAME"
pg_restore \
	--dbname="$RESTORE_DATABASE_URL" \
	--clean \
	--if-exists \
	"backups/aura_YYYYMMDDTHHMMSSZ.dump"
```

Plain SQL backup:

```bash
createdb "$RESTORE_DATABASE_NAME"
psql "$RESTORE_DATABASE_URL" < "backups/aura_YYYYMMDDTHHMMSSZ.sql"
```

After restore:

```bash
migrate -path migrations -database "$RESTORE_DATABASE_URL" up
go test ./...
```

Then start the API against the restored database and verify:

```bash
curl -fsS http://localhost:8080/health
```

## Rollback Notes

1. Prefer application rollback first: redeploy the previous image with the same database.
2. If a migration introduced bad data or incompatible schema changes, restore the latest pre-release backup into a new database and switch the application to it.
3. Do not run destructive down migrations against production without a current backup and an explicit recovery plan.
4. Re-run `go run ./cmd/bootstrap-admin` only if the restored database has no active `SUPER_ADMIN`.
