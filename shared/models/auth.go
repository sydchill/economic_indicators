package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:auth.users,alias:u"`

	ID            uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Email         string    `bun:"email,notnull,unique"`
	EmailVerified bool      `bun:"email_verified,notnull,default:false"`
	Status        string    `bun:"status,notnull,default:'active'"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:now()"`
}

type Credential struct {
	bun.BaseModel `bun:"table:auth.credentials,alias:c"`

	UserID            uuid.UUID `bun:"user_id,pk,type:uuid"`
	PasswordHash      string    `bun:"password_hash,notnull"`
	PasswordChangedAt time.Time `bun:"password_changed_at,notnull,default:now()"`
}

type OAuthAccount struct {
	bun.BaseModel `bun:"table:auth.oauth_accounts,alias:oa"`

	ID             uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID         uuid.UUID `bun:"user_id,type:uuid,notnull"`
	Provider       string    `bun:"provider,notnull"`
	ProviderUserID string    `bun:"provider_user_id,notnull"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:now()"`
}

type Session struct {
	bun.BaseModel `bun:"table:auth.sessions,alias:s"`

	ID        uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID    uuid.UUID  `bun:"user_id,type:uuid,notnull"`
	TokenHash string     `bun:"token_hash,notnull,unique"`
	ExpiresAt time.Time  `bun:"expires_at,notnull"`
	RevokedAt *time.Time `bun:"revoked_at"`
	CreatedAt time.Time  `bun:"created_at,notnull,default:now()"`
}

type PasswordResetToken struct {
	bun.BaseModel `bun:"table:auth.password_reset_tokens,alias:prt"`

	ID        uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID    uuid.UUID  `bun:"user_id,type:uuid,notnull"`
	TokenHash string     `bun:"token_hash,notnull,unique"`
	ExpiresAt time.Time  `bun:"expires_at,notnull"`
	UsedAt    *time.Time `bun:"used_at"`
	CreatedAt time.Time  `bun:"created_at,notnull,default:now()"`
}
