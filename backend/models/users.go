package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Profile struct {
	bun.BaseModel `bun:"table:users.profiles,alias:p"`

	UserID      uuid.UUID `bun:"user_id,pk,type:uuid"`
	DisplayName *string   `bun:"display_name"`
	AvatarURL   *string   `bun:"avatar_url"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()"`
}
