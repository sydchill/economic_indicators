package auth

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
)

// GoogleClaims is the subset of Google ID token claims we persist.
type GoogleClaims struct {
	Subject       string // stable Google user ID ("sub" claim)
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

// verifyGoogleIDToken validates the token's signature, issuer, audience,
// expiry, and iat. It returns a panic-safe error for any malformed input.
func verifyGoogleIDToken(ctx context.Context, rawIDToken, clientID string) (claims *GoogleClaims, err error) {
	if clientID == "" {
		return nil, fmt.Errorf("google client id is not configured")
	}

	// idtoken.Validate has been observed to panic on some malformed
	// signatures (e.g. a truncated ES256 token) before it can return
	// an error. No request may crash the handler.
	defer func() {
		if r := recover(); r != nil {
			claims = nil
			err = ErrGoogleTokenInvalid
		}
	}()

	payload, err := idtoken.Validate(ctx, rawIDToken, clientID)
	if err != nil {
		return nil, ErrGoogleTokenInvalid
	}

	// idtoken.Validate checks aud against clientID. If the token carries
	// multiple audiences, the OIDC spec requires azp == our client_id.
	// Reject tokens that were minted for a different client in the same set.
	if azp, ok := payload.Claims["azp"].(string); ok && azp != "" && azp != clientID {
		return nil, ErrGoogleTokenInvalid
	}

	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	if email == "" {
		return nil, ErrGoogleTokenInvalid
	}
	if !emailVerified {
		return nil, ErrGoogleEmailUnverified
	}

	return &GoogleClaims{
		Subject:       payload.Subject,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		Picture:       picture,
	}, nil
}
