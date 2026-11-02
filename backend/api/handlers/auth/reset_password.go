package auth

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	userpkg "github.com/sydchill/economic_indicators/models"
	"github.com/uptrace/bun"
)

const resetTokenTTL = 30 * time.Minute // OWASP recommends ≤ 30 minutes[reference:3]

// RequestPasswordReset always returns nil, even if the email is unknown.
// That guarantees the handler can safely return 204 for every request.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = normalizeEmail(email)

	usr := new(userpkg.User)
	err := s.db.NewSelect().Model(usr).
		Where("u.email = ?", email).
		Where("u.status = ?", "active").
		Scan(ctx)
	if err != nil {
		// Do not propagate "not found" — the caller must not know.
		return nil
	}

	raw, hash, err := GenerateSessionToken() // same 32-byte random + SHA-256 scheme
	if err != nil {
		return err
	}

	token := &userpkg.PasswordResetToken{
		UserID:    usr.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(resetTokenTTL),
	}
	if _, err := s.db.NewInsert().Model(token).Exec(ctx); err != nil {
		return err
	}

	// --- email delivery (development stub) ---
	// Replace with a real mailer in a later component.
	// The raw token is logged only in non-production builds.
	log.Printf("[DEV] password reset token for %s: %s", email, raw)

	return nil
}

// ResetPassword validates the new password, atomically consumes the token,
// updates the credential, revokes all sessions, and invalidates sibling tokens.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err // ErrWeakPassword — checked before consuming the token
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	tokenHash := HashSessionToken(rawToken) // SHA-256 of the raw token

	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// 1. Atomic consume: only one concurrent request can claim this row.
		var userID uuid.UUID
		err := tx.NewRaw(`
			UPDATE auth.password_reset_tokens
			SET used_at = NOW()
			WHERE token_hash = ?
			  AND used_at IS NULL
			  AND expires_at > NOW()
			RETURNING user_id
		`, tokenHash).Scan(ctx, &userID)
		if err != nil {
			// No row returned → invalid, expired, or already consumed.
			return ErrInvalidResetToken
		}

		// 2. Update the password credential.
		if _, err := tx.NewUpdate().
			TableExpr("auth.credentials").
			Set("password_hash = ?", newHash).
			Set("password_changed_at = NOW()").
			Where("user_id = ?", userID).
			Exec(ctx); err != nil {
			return err
		}

		// 3. Revoke every active session for this user.
		if _, err := tx.NewUpdate().
			TableExpr("auth.sessions").
			Set("revoked_at = NOW()").
			Where("user_id = ?", userID).
			Where("revoked_at IS NULL").
			Exec(ctx); err != nil {
			return err
		}

		// 4. Invalidate any other outstanding reset tokens for the same user.
		if _, err := tx.NewUpdate().
			TableExpr("auth.password_reset_tokens").
			Set("used_at = NOW()").
			Where("user_id = ?", userID).
			Where("used_at IS NULL").
			Where("id <> (SELECT id FROM auth.password_reset_tokens WHERE token_hash = ?)", tokenHash).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}

func normalizeEmail(email string) string {
	// CITEXT already makes the column case-insensitive, but trimming
	// avoids accidental whitespace mismatches from HTTP payloads.
	return strings.TrimSpace(email)
}
