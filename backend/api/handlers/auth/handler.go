package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.Get("/me", h.Me)
	return r
}

// ---------------------------------------------------------------

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	usr, err := h.svc.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrWeakPassword):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrEmailTaken):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "registration failed")
		}
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{
		"user_id": usr.ID,
		"email":   usr.Email,
	})
}

// ---------------------------------------------------------------

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	usr, session, raw, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		// Never leak whether the email exists — both branches return 401.
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrAccountInactive) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"session_token": raw,
		"expires_at":    session.ExpiresAt,
		"user": map[string]any{
			"id":             usr.ID,
			"email":          usr.Email,
			"email_verified": usr.EmailVerified,
		},
	})
}

// ---------------------------------------------------------------

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	_ = h.svc.Logout(r.Context(), token)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	sc, err := h.svc.ResolveSession(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}

	body := map[string]any{
		"id":             sc.User.ID,
		"email":          sc.User.Email,
		"email_verified": sc.User.EmailVerified,
		"status":         sc.User.Status,
		"created_at":     sc.User.CreatedAt,
	}
	if sc.Profile != nil {
		body["display_name"] = sc.Profile.DisplayName
		body["avatar_url"] = sc.Profile.AvatarURL
	}
	WriteJSON(w, http.StatusOK, body)
}

// ---------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(h, "Bearer "), true
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}
