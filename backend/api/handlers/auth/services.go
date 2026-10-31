package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	userpkg "github.com/sydchill/economic_indicators/models"
	"github.com/uptrace/bun"
)

type Service struct {
	db *bun.DB
}

func NewService(db *bun.DB) *Service {
	return &Service{db: db}
}

type SessionContext struct {
	User    *userpkg.User
	Profile *userpkg.Profile
}

// ---------------------------------------------------------------
// Register
// ---------------------------------------------------------------

func (s *Service) Register(ctx context.Context, email, password, displayName string) (*userpkg.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	usr := &userpkg.User{
		Email:         email,
		EmailVerified: false,
		Status:        "active",
	}

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(usr).
			Returning("id, created_at, updated_at").
			Exec(ctx); err != nil {
			return err
		}
		cred := &userpkg.Credential{UserID: usr.ID, PasswordHash: hash}
		if _, err := tx.NewInsert().Model(cred).Exec(ctx); err != nil {
			return err
		}
		profile := &userpkg.Profile{UserID: usr.ID}
		if displayName != "" {
			name := displayName
			profile.DisplayName = &name
		}
		if _, err := tx.NewInsert().Model(profile).Exec(ctx); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if isEmailConflict(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return usr, nil
}

// ---------------------------------------------------------------
// Login
// ---------------------------------------------------------------

func (s *Service) Login(ctx context.Context, email, password string) (*userpkg.User, *userpkg.Session, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	usr := new(userpkg.User)
	err := s.db.NewSelect().Model(usr).Where("u.email = ?", email).Scan(ctx)
	if err != nil {
		// Constant-time-ish: burn an argon2 verify against a dummy hash
		// so "user not found" and "wrong password" take the same time.
		_, _ = VerifyPassword(password, DummyHash)
		return nil, nil, "", ErrInvalidCredentials
	}

	cred := new(userpkg.Credential)
	if err := s.db.NewSelect().Model(cred).Where("c.user_id = ?", usr.ID).Scan(ctx); err != nil {
		_, _ = VerifyPassword(password, DummyHash)
		return nil, nil, "", ErrInvalidCredentials
	}

	ok, err := VerifyPassword(password, cred.PasswordHash)
	if err != nil || !ok {
		return nil, nil, "", ErrInvalidCredentials
	}
	if usr.Status != "active" {
		return nil, nil, "", ErrAccountInactive
	}

	raw, hash, err := GenerateSessionToken()
	if err != nil {
		return nil, nil, "", err
	}

	session := &userpkg.Session{
		UserID:    usr.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if _, err := s.db.NewInsert().Model(session).
		Returning("id, created_at").
		Exec(ctx); err != nil {
		return nil, nil, "", err
	}

	return usr, session, raw, nil
}

// ---------------------------------------------------------------
// Logout (idempotent — no error if token was already revoked/unknown)
// ---------------------------------------------------------------

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	hash := HashSessionToken(rawToken)
	_, err := s.db.NewUpdate().
		TableExpr("auth.sessions").
		Set("revoked_at = NOW()").
		Where("token_hash = ?", hash).
		Where("revoked_at IS NULL").
		Exec(ctx)
	return err
}

// ---------------------------------------------------------------
// ResolveSession — used by /me and any protected route
// ---------------------------------------------------------------

func (s *Service) ResolveSession(ctx context.Context, rawToken string) (*SessionContext, error) {
	hash := HashSessionToken(rawToken)

	session := new(userpkg.Session)
	err := s.db.NewSelect().Model(session).
		Where("s.token_hash = ?", hash).
		Where("s.revoked_at IS NULL").
		Where("s.expires_at > NOW()").
		Scan(ctx)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	usr := new(userpkg.User)
	if err := s.db.NewSelect().Model(usr).
		Where("u.id = ?", session.UserID).
		Scan(ctx); err != nil {
		return nil, ErrSessionNotFound
	}

	profile := new(userpkg.Profile)
	if err := s.db.NewSelect().Model(profile).
		Where("p.user_id = ?", usr.ID).
		Scan(ctx); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &SessionContext{User: usr, Profile: profile}, nil
}

// ---------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------

func isEmailConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return strings.Contains(pgErr.ConstraintName, "email")
	}
	return false
}
