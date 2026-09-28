package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// tableSpec is just a name container; per-table logic lives in dedicated
// migrateXxx functions because each has its own type-conversion and FK-remap
// concerns. The list defines FK-safe ordering.
type tableSpec struct {
	name      string
	migrateFn func(ctx context.Context, src *sql.DB, tx pgx.Tx, plans map[int64]int64) (int, error)
}

// allTables returns every table we migrate, in FK-safe order. inspect_exchanges
// comes last and is intentionally placed after the main transaction commit.
func allTables() []tableSpec {
	return []tableSpec{
		{"plans", nil}, // handled inline by migratePlans before this loop
		{"users", migrateUsers},
		{"invite_codes", migrateInviteCodes},
		{"reserved_domains", migrateReservedDomains},
		{"sessions", migrateSessions},
		{"api_tokens", migrateAPITokens},
		{"totp_secrets", migrateTOTPSecrets},
		{"audit_logs", migrateAuditLogs},
		{"user_bundles", migrateUserBundles},
		{"user_history", migrateUserHistory},
		{"user_settings", migrateUserSettings},
		{"custom_domains", migrateCustomDomains},
		{"tls_certificates", migrateTLSCertificates},
		{"subscriptions", migrateSubscriptions},
		{"payments", migratePayments},
		{"inspect_exchanges", migrateInspectExchanges},
	}
}

// migrateTable dispatches to the per-table function. Plans map is supplied for
// FK-ремап of plan_id/next_plan_id columns.
func migrateTable(ctx context.Context, src *sql.DB, tx pgx.Tx, spec tableSpec, plans map[int64]int64) (int, error) {
	if spec.migrateFn == nil {
		return 0, fmt.Errorf("table %s has no migrate function", spec.name)
	}
	return spec.migrateFn(ctx, src, tx, plans)
}

// ---------------------------------------------------------------------------
// plans — UPSERT by slug and build oldID→newID map for FK remap.
// ---------------------------------------------------------------------------

func migratePlans(ctx context.Context, src *sql.DB, tx pgx.Tx) (map[int64]int64, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, slug, name, COALESCE(price,0), max_tunnels, max_domains,
		       COALESCE(max_custom_domains,0), max_tokens, max_tunnels_per_token,
		       CAST(inspector_enabled AS INTEGER),
		       CAST(COALESCE(is_public,0) AS INTEGER),
		       CAST(COALESCE(is_recommended,0) AS INTEGER),
		       COALESCE(bandwidth_mbps,0), COALESCE(rate_limit_tcp,0),
		       COALESCE(rate_limit_udp,0), COALESCE(rate_limit_http,0),
		       COALESCE(creem_product_id,'')
		FROM plans
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query sqlite plans: %w", err)
	}
	defer rows.Close()

	planMap := map[int64]int64{}
	for rows.Next() {
		var (
			oldID, maxT, maxD, maxCD, maxTok, maxTPT, bwMbps, rlTCP, rlUDP, rlHTTP int64
			slug, name, creemPID                                                   string
			price                                                                  float64
			inspector, isPublic, isRec                                             int64
		)
		if err := rows.Scan(&oldID, &slug, &name, &price, &maxT, &maxD, &maxCD,
			&maxTok, &maxTPT, &inspector, &isPublic, &isRec,
			&bwMbps, &rlTCP, &rlUDP, &rlHTTP, &creemPID); err != nil {
			return nil, fmt.Errorf("scan plans: %w", err)
		}

		var newID int64
		// UPSERT: if slug exists (seeded by goose), keep existing id and update editable fields.
		err := tx.QueryRow(ctx, `
			INSERT INTO plans (slug, name, price, max_tunnels, max_domains, max_custom_domains,
			                   max_tokens, max_tunnels_per_token, inspector_enabled,
			                   is_public, is_recommended, bandwidth_mbps,
			                   rate_limit_tcp, rate_limit_udp, rate_limit_http, creem_product_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`,
			slug, name, price, maxT, maxD, maxCD, maxTok, maxTPT,
			inspector != 0, isPublic != 0, isRec != 0,
			bwMbps, rlTCP, rlUDP, rlHTTP, creemPID,
		).Scan(&newID)
		if err != nil {
			return nil, fmt.Errorf("upsert plan %s: %w", slug, err)
		}
		planMap[oldID] = newID
	}
	return planMap, rows.Err()
}

// ---------------------------------------------------------------------------
// users — phone='' → NULL (for partial unique idx), plan_id remap.
// ---------------------------------------------------------------------------

func migrateUsers(ctx context.Context, src *sql.DB, tx pgx.Tx, plans map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, phone, password_hash, display_name,
		       CAST(is_admin AS INTEGER), CAST(is_active AS INTEGER),
		       created_at, last_login_at, github_id, email, avatar_url, google_id,
		       plan_id, first_tunnel_at
		FROM users
		ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			id                                                                       int64
			phone, passwordHash, displayName, email, avatarURL, googleID, createdAt sql.NullString
			lastLoginAt, firstTunnelAt                                              sql.NullString
			isAdmin, isActive                                                       sql.NullInt64
			githubID, planID                                                        sql.NullInt64
		)
		if err := rows.Scan(&id, &phone, &passwordHash, &displayName, &isAdmin, &isActive,
			&createdAt, &lastLoginAt, &githubID, &email, &avatarURL, &googleID,
			&planID, &firstTunnelAt); err != nil {
			return count, fmt.Errorf("scan users: %w", err)
		}

		createdTime, _ := parseSQLiteTime(createdAt)
		lastLoginTime, _ := parseSQLiteTime(lastLoginAt)
		firstTunnelTime, _ := parseSQLiteTime(firstTunnelAt)

		var pgPlanID any
		if planID.Valid {
			if newID, ok := plans[planID.Int64]; ok {
				pgPlanID = newID
			}
		}

		// password_hash has DEFAULT '' — keep empty string, not NULL.
		ph := ""
		if passwordHash.Valid {
			ph = passwordHash.String
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO users (id, phone, password_hash, display_name, is_admin, is_active,
			                   created_at, last_login_at, github_id, email, avatar_url,
			                   google_id, plan_id, first_tunnel_at)
			VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, NOW()), $8, $9, $10, $11, $12, $13, $14)`,
			id,
			nullableString(phone), // '' → NULL for partial unique
			ph,
			nullableStringKeepEmpty(displayName),
			boolFromInt(isAdmin),
			isActive.Valid && isActive.Int64 != 0 || !isActive.Valid, // DEFAULT TRUE if NULL
			createdTime,
			lastLoginTime,
			nullableInt64(githubID),
			nullableStringKeepEmpty(email),
			nullableStringKeepEmpty(avatarURL),
			nullableStringKeepEmpty(googleID),
			pgPlanID,
			firstTunnelTime,
		)
		if err != nil {
			return count, fmt.Errorf("insert user id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// invite_codes — straight copy, FK to users.id preserved.
// ---------------------------------------------------------------------------

func migrateInviteCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, code, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invite_codes ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			id                                int64
			code                              string
			createdBy, usedBy                 sql.NullInt64
			usedAt, expiresAt, createdAt      sql.NullString
		)
		if err := rows.Scan(&id, &code, &createdBy, &usedBy, &usedAt, &expiresAt, &createdAt); err != nil {
			return count, err
		}
		usedT, _ := parseSQLiteTime(usedAt)
		expT, _ := parseSQLiteTime(expiresAt)
		createdT, _ := parseSQLiteTime(createdAt)
		_, err := tx.Exec(ctx, `
			INSERT INTO invite_codes (id, code, created_by_user_id, used_by_user_id, used_at, expires_at, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,COALESCE($7, NOW()))`,
			id, code, nullableInt64(createdBy), nullableInt64(usedBy), usedT, expT, createdT,
		)
		if err != nil {
			return count, fmt.Errorf("invite_codes id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// reserved_domains
// ---------------------------------------------------------------------------

func migrateReservedDomains(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `SELECT id, user_id, subdomain, created_at FROM reserved_domains ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID int64
		var subdomain string
		var createdAt sql.NullString
		if err := rows.Scan(&id, &userID, &subdomain, &createdAt); err != nil {
			return count, err
		}
		createdT, _ := parseSQLiteTime(createdAt)
		if _, err := tx.Exec(ctx,
			`INSERT INTO reserved_domains (id, user_id, subdomain, created_at) VALUES ($1,$2,$3, COALESCE($4, NOW()))`,
			id, userID, subdomain, createdT); err != nil {
			return count, fmt.Errorf("reserved_domains id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// sessions
// ---------------------------------------------------------------------------

func migrateSessions(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
		FROM sessions ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID int64
		var tokenHash string
		var userAgent, ipAddr sql.NullString
		var expiresAt, createdAt sql.NullString
		if err := rows.Scan(&id, &userID, &tokenHash, &userAgent, &ipAddr, &expiresAt, &createdAt); err != nil {
			return count, err
		}
		expT, _ := parseSQLiteTime(expiresAt)
		createdT, _ := parseSQLiteTime(createdAt)
		if expT == nil {
			// expires_at is NOT NULL in PG; skip the session rather than fail
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,COALESCE($7, NOW()))`,
			id, userID, tokenHash, nullableStringKeepEmpty(userAgent), nullableStringKeepEmpty(ipAddr),
			expT, createdT); err != nil {
			return count, fmt.Errorf("sessions id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// api_tokens — TEXT JSON → JSONB.
// ---------------------------------------------------------------------------

func migrateAPITokens(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, token_hash, name, allowed_subdomains, max_tunnels,
		       allowed_ips, last_used_at, created_at
		FROM api_tokens ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID int64
		var tokenHash, name string
		var allowedSub, allowedIPs sql.NullString
		var maxTunnels sql.NullInt64
		var lastUsedAt, createdAt sql.NullString
		if err := rows.Scan(&id, &userID, &tokenHash, &name, &allowedSub, &maxTunnels,
			&allowedIPs, &lastUsedAt, &createdAt); err != nil {
			return count, err
		}
		lastUsedT, _ := parseSQLiteTime(lastUsedAt)
		createdT, _ := parseSQLiteTime(createdAt)
		max := int64(10)
		if maxTunnels.Valid {
			max = maxTunnels.Int64
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO api_tokens (id, user_id, token_hash, name, allowed_subdomains, max_tunnels,
			                       allowed_ips, last_used_at, created_at)
			VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7::jsonb,$8,COALESCE($9, NOW()))`,
			id, userID, tokenHash, name,
			jsonbOrDefault(allowedSub, "[]"), max,
			jsonbOrDefault(allowedIPs, "[]"),
			lastUsedT, createdT); err != nil {
			return count, fmt.Errorf("api_tokens id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// totp_secrets — backup_codes TEXT(json) → JSONB nullable.
// ---------------------------------------------------------------------------

func migrateTOTPSecrets(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, secret_encrypted, CAST(is_enabled AS INTEGER), backup_codes, created_at
		FROM totp_secrets ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID int64
		var secret string
		var isEnabled sql.NullInt64
		var backupCodes, createdAt sql.NullString
		if err := rows.Scan(&id, &userID, &secret, &isEnabled, &backupCodes, &createdAt); err != nil {
			return count, err
		}
		createdT, _ := parseSQLiteTime(createdAt)
		if _, err := tx.Exec(ctx, `
			INSERT INTO totp_secrets (id, user_id, secret_encrypted, is_enabled, backup_codes, created_at)
			VALUES ($1,$2,$3,$4, NULLIF($5,'')::jsonb, COALESCE($6, NOW()))`,
			id, userID, secret, boolFromInt(isEnabled), jsonbNullable(backupCodes), createdT); err != nil {
			return count, fmt.Errorf("totp_secrets id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// audit_logs — details TEXT → JSONB nullable.
// ---------------------------------------------------------------------------

func migrateAuditLogs(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, action, details, ip_address, created_at
		FROM audit_logs ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id int64
		var userID sql.NullInt64
		var action string
		var details, ipAddr, createdAt sql.NullString
		if err := rows.Scan(&id, &userID, &action, &details, &ipAddr, &createdAt); err != nil {
			return count, err
		}
		createdT, _ := parseSQLiteTime(createdAt)
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_logs (id, user_id, action, details, ip_address, created_at)
			VALUES ($1,$2,$3,NULLIF($4,'')::jsonb,$5,COALESCE($6, NOW()))`,
			id, nullableInt64(userID), action, jsonbNullable(details),
			nullableStringKeepEmpty(ipAddr), createdT); err != nil {
			return count, fmt.Errorf("audit_logs id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// user_bundles
// ---------------------------------------------------------------------------

func migrateUserBundles(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, name, type, local_port, subdomain, remote_port,
		       CAST(auto_connect AS INTEGER), created_at, updated_at
		FROM user_bundles ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID, localPort int64
		var name, btype string
		var subdomain sql.NullString
		var remotePort sql.NullInt64
		var autoConnect sql.NullInt64
		var createdAt, updatedAt sql.NullString
		if err := rows.Scan(&id, &userID, &name, &btype, &localPort, &subdomain, &remotePort,
			&autoConnect, &createdAt, &updatedAt); err != nil {
			return count, err
		}
		createdT, _ := parseSQLiteTime(createdAt)
		updatedT, _ := parseSQLiteTime(updatedAt)
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_bundles (id, user_id, name, type, local_port, subdomain, remote_port,
			                          auto_connect, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,COALESCE($9, NOW()),COALESCE($10, NOW()))`,
			id, userID, name, btype, localPort, nullableStringKeepEmpty(subdomain),
			nullableInt64(remotePort), boolFromInt(autoConnect), createdT, updatedT); err != nil {
			return count, fmt.Errorf("user_bundles id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// user_history
// ---------------------------------------------------------------------------

func migrateUserHistory(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, bundle_name, tunnel_type, local_port, remote_addr, url,
		       connected_at, disconnected_at, bytes_sent, bytes_received
		FROM user_history ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID, localPort int64
		var bundleName, tunnelType, remoteAddr, url sql.NullString
		var connAt, disconnAt sql.NullString
		var bytesSent, bytesRecv sql.NullInt64
		if err := rows.Scan(&id, &userID, &bundleName, &tunnelType, &localPort, &remoteAddr, &url,
			&connAt, &disconnAt, &bytesSent, &bytesRecv); err != nil {
			return count, err
		}
		connT, _ := parseSQLiteTime(connAt)
		disT, _ := parseSQLiteTime(disconnAt)
		if connT == nil {
			continue // connected_at is NOT NULL
		}
		tType := ""
		if tunnelType.Valid {
			tType = tunnelType.String
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_history (id, user_id, bundle_name, tunnel_type, local_port,
			                          remote_addr, url, connected_at, disconnected_at,
			                          bytes_sent, bytes_received)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			id, userID, nullableStringKeepEmpty(bundleName), tType, localPort,
			nullableStringKeepEmpty(remoteAddr), nullableStringKeepEmpty(url),
			connT, disT,
			bytesSent.Int64, bytesRecv.Int64); err != nil {
			return count, fmt.Errorf("user_history id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// user_settings — composite PK (user_id, key)
// ---------------------------------------------------------------------------

func migrateUserSettings(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `SELECT user_id, key, value, updated_at FROM user_settings`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var userID int64
		var key, value string
		var updatedAt sql.NullString
		if err := rows.Scan(&userID, &key, &value, &updatedAt); err != nil {
			return count, err
		}
		updT, _ := parseSQLiteTime(updatedAt)
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_settings (user_id, key, value, updated_at)
			VALUES ($1,$2,$3,COALESCE($4, NOW()))
			ON CONFLICT (user_id, key) DO NOTHING`,
			userID, key, value, updT); err != nil {
			return count, fmt.Errorf("user_settings user=%d key=%s: %w", userID, key, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// custom_domains
// ---------------------------------------------------------------------------

func migrateCustomDomains(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, domain, target_subdomain, CAST(verified AS INTEGER), verified_at, created_at
		FROM custom_domains ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID int64
		var domain, target string
		var verified sql.NullInt64
		var verifiedAt, createdAt sql.NullString
		if err := rows.Scan(&id, &userID, &domain, &target, &verified, &verifiedAt, &createdAt); err != nil {
			return count, err
		}
		vT, _ := parseSQLiteTime(verifiedAt)
		cT, _ := parseSQLiteTime(createdAt)
		if _, err := tx.Exec(ctx, `
			INSERT INTO custom_domains (id, user_id, domain, target_subdomain, verified, verified_at, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,COALESCE($7, NOW()))`,
			id, userID, domain, target, boolFromInt(verified), vT, cT); err != nil {
			return count, fmt.Errorf("custom_domains id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// tls_certificates — BLOB → BYTEA
// ---------------------------------------------------------------------------

func migrateTLSCertificates(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, domain, cert_pem, key_pem, expires_at, issued_at, created_at
		FROM tls_certificates ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id int64
		var domain string
		var certPEM, keyPEM []byte
		var expiresAt, issuedAt, createdAt sql.NullString
		if err := rows.Scan(&id, &domain, &certPEM, &keyPEM, &expiresAt, &issuedAt, &createdAt); err != nil {
			return count, err
		}
		expT, _ := parseSQLiteTime(expiresAt)
		issT, _ := parseSQLiteTime(issuedAt)
		cT, _ := parseSQLiteTime(createdAt)
		if expT == nil || issT == nil {
			continue // both are NOT NULL in PG
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO tls_certificates (id, domain, cert_pem, key_pem, expires_at, issued_at, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,COALESCE($7, NOW()))`,
			id, domain, certPEM, keyPEM, expT, issT, cT); err != nil {
			return count, fmt.Errorf("tls_certificates id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// subscriptions — plan_id/next_plan_id remap. Drops legacy robokassa/stripe cols.
// ---------------------------------------------------------------------------

func migrateSubscriptions(ctx context.Context, src *sql.DB, tx pgx.Tx, plans map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, plan_id, next_plan_id, status, CAST(recurring AS INTEGER),
		       current_period_start, current_period_end,
		       yookassa_payment_method_id, creem_customer_id, creem_subscription_id,
		       created_at, updated_at
		FROM subscriptions ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID, planID int64
		var nextPlanID sql.NullInt64
		var status string
		var recurring sql.NullInt64
		var periodStart, periodEnd, createdAt, updatedAt sql.NullString
		var yookassaPMID, creemCustID, creemSubID sql.NullString
		if err := rows.Scan(&id, &userID, &planID, &nextPlanID, &status, &recurring,
			&periodStart, &periodEnd, &yookassaPMID, &creemCustID, &creemSubID,
			&createdAt, &updatedAt); err != nil {
			return count, err
		}
		mappedPlan, ok := plans[planID]
		if !ok {
			// Plan disappeared — fall back to free (slug-based lookup would be nicer)
			continue
		}
		var mappedNext any
		if nextPlanID.Valid {
			if np, ok := plans[nextPlanID.Int64]; ok {
				mappedNext = np
			}
		}
		psT, _ := parseSQLiteTime(periodStart)
		peT, _ := parseSQLiteTime(periodEnd)
		cT, _ := parseSQLiteTime(createdAt)
		uT, _ := parseSQLiteTime(updatedAt)
		// recurring defaults to TRUE in PG
		rec := !recurring.Valid || recurring.Int64 != 0

		if _, err := tx.Exec(ctx, `
			INSERT INTO subscriptions (id, user_id, plan_id, next_plan_id, status, recurring,
			                            current_period_start, current_period_end,
			                            yookassa_payment_method_id, creem_customer_id, creem_subscription_id,
			                            created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,COALESCE($12, NOW()),COALESCE($13, NOW()))`,
			id, userID, mappedPlan, mappedNext, status, rec,
			psT, peT,
			nullableStringKeepEmpty(yookassaPMID), nullableStringKeepEmpty(creemCustID),
			nullableStringKeepEmpty(creemSubID), cT, uT); err != nil {
			return count, fmt.Errorf("subscriptions id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// payments — drops legacy robokassa_data column.
// ---------------------------------------------------------------------------

func migratePayments(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, user_id, subscription_id, invoice_id, amount, status,
		       CAST(is_recurring AS INTEGER),
		       yookassa_data, provider, provider_data, created_at
		FROM payments ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID, invoiceID int64
		var subID sql.NullInt64
		var amount float64
		var status, provider string
		var isRecurring sql.NullInt64
		var yookassaData, providerData sql.NullString
		var createdAt sql.NullString
		if err := rows.Scan(&id, &userID, &subID, &invoiceID, &amount, &status, &isRecurring,
			&yookassaData, &provider, &providerData, &createdAt); err != nil {
			return count, err
		}
		cT, _ := parseSQLiteTime(createdAt)
		if _, err := tx.Exec(ctx, `
			INSERT INTO payments (id, user_id, subscription_id, invoice_id, amount, status, is_recurring,
			                     yookassa_data, provider, provider_data, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,COALESCE($11, NOW()))`,
			id, userID, nullableInt64(subID), invoiceID, amount, status, boolFromInt(isRecurring),
			nullableStringKeepEmpty(yookassaData), provider, nullableStringKeepEmpty(providerData), cT); err != nil {
			return count, fmt.Errorf("payments id=%d: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// ---------------------------------------------------------------------------
// inspect_exchanges — large BYTEA payloads. Default-skip via --skip flag.
// ---------------------------------------------------------------------------

func migrateInspectExchanges(ctx context.Context, src *sql.DB, tx pgx.Tx, _ map[int64]int64) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, tunnel_id, user_id, trace_id, replay_ref, timestamp, duration_ns,
		       method, path, host, request_headers, request_body, request_body_size,
		       response_headers, response_body, response_body_size, status_code,
		       remote_addr, created_at
		FROM inspect_exchanges ORDER BY rowid`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, tunnelID, method, path, host string
		var userID int64
		var traceID, replayRef, reqHeaders, respHeaders, remoteAddr sql.NullString
		var timestamp, createdAt sql.NullString
		var durationNS, reqBodySize, respBodySize, statusCode int64
		var reqBody, respBody []byte
		if err := rows.Scan(&id, &tunnelID, &userID, &traceID, &replayRef, &timestamp, &durationNS,
			&method, &path, &host, &reqHeaders, &reqBody, &reqBodySize,
			&respHeaders, &respBody, &respBodySize, &statusCode,
			&remoteAddr, &createdAt); err != nil {
			return count, err
		}
		tsT, _ := parseSQLiteTime(timestamp)
		cT, _ := parseSQLiteTime(createdAt)
		if tsT == nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO inspect_exchanges (id, tunnel_id, user_id, trace_id, replay_ref, timestamp, duration_ns,
			                              method, path, host, request_headers, request_body, request_body_size,
			                              response_headers, response_body, response_body_size, status_code,
			                              remote_addr, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::jsonb,$12,$13,NULLIF($14,'')::jsonb,$15,$16,$17,$18,COALESCE($19, NOW()))
			ON CONFLICT (id) DO NOTHING`,
			id, tunnelID, userID, nullableStringKeepEmpty(traceID), nullableStringKeepEmpty(replayRef),
			tsT, durationNS, method, path, host,
			jsonbNullable(reqHeaders), blobOrNil(reqBody), reqBodySize,
			jsonbNullable(respHeaders), blobOrNil(respBody), respBodySize, statusCode,
			nullableStringKeepEmpty(remoteAddr), cT); err != nil {
			return count, fmt.Errorf("inspect_exchanges id=%s: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}
