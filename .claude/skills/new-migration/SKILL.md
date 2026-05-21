---
name: new-migration
description: Scaffold a new Postgres schema migration for fxTunnel (goose + sqlc workflow). Use when adding/altering a table or column, adding a query, or otherwise changing the database schema.
disable-model-invocation: true
---

# new-migration

Scaffold and wire a new database change end-to-end. fxTunnel uses **goose** (embedded migrations, applied automatically on server start) for schema and **sqlc** (`pgx/v5`) for typed queries. Generated code in `internal/server/database/sqlc/` is read-only (a hook blocks edits).

All paths are relative to `internal/server/database/`.

## Steps

1. **Pick the next migration number.** List `migrations/` and increment the highest `NNNNN_` prefix (zero-padded to 5 digits). Name it `NNNNN_short_snake_case.sql`.

2. **Write the migration** with goose annotations — both directions are required:
   ```sql
   -- +goose Up
   ALTER TABLE <table> ADD COLUMN <col> <type> NOT NULL DEFAULT <val>;

   -- +goose Down
   ALTER TABLE <table> DROP COLUMN <col>;
   ```
   For multi-statement or function bodies use `-- +goose StatementBegin` / `-- +goose StatementEnd`. Keep `Down` a true inverse. Backfills/seed updates go in `Up` after the DDL (see `00005_add_udp_enabled.sql`, `00006`, `00007` for patterns).

3. **Add/adjust queries** in the matching `queries/*.sql` file (e.g. `plans.sql`, `users.sql`). Use sqlc annotations: `-- name: GetX :one|:many|:exec`. Reference columns you just added.

4. **Regenerate sqlc** from the database dir:
   ```bash
   cd internal/server/database && /home/fxcode/go/bin/sqlc generate
   ```
   This rewrites `sqlc/*.sql.go`, `models.go`, `querier.go`. Never hand-edit them.

5. **Wire the repository.** Expose the new query through the matching `pg_*_repo.go` wrapper (these adapt the generated querier to the `store/` interfaces consumed by handlers). Update the `store/` interface if the surface changed.

6. **Verify:**
   ```bash
   go build ./... && go vet ./internal/server/database/...
   ```
   Migrations apply on next server boot via `goose.Up`; there is no separate migrate command. If you have a local Postgres, smoke-test by starting the server against it.

## Notes

- Migrations are embedded via `//go:embed migrations/*.sql` in `database.go` — a new `.sql` file is picked up automatically, no registration needed.
- Production/staging is Postgres; the default `configs/server.yaml` points at a legacy SQLite path, so local schema testing needs a real Postgres DSN (`FXTUNNEL_DATABASE_DSN`).
- Commit with a `feat`/`fix` scope, e.g. `feat(db): add <col> to <table>`.
