package rest

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
)

type Handler struct {
	register *usecase.RegisterUseCase
	login    *usecase.LoginUseCase
	refresh  *usecase.RefreshUseCase
	tokenSvc *auth.TokenService
}

func NewHandler(
	register *usecase.RegisterUseCase,
	login *usecase.LoginUseCase,
	refresh *usecase.RefreshUseCase,
	tokenSvc *auth.TokenService,
) *Handler {
	return &Handler{
		register: register,
		login:    login,
		refresh:  refresh,
		tokenSvc: tokenSvc,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.register.Execute(r.Context(), usecase.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already taken")
		case errors.Is(err, auth.ErrPasswordTooShort):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, tokenResponse{
		AccessToken:  string(out.Tokens.Access),
		RefreshToken: string(out.Tokens.Refresh),
		ExpiresAt:    out.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	out, err := h.login.Execute(r.Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound),
			errors.Is(err, auth.ErrPasswordInvalid):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  string(out.Tokens.Access),
		RefreshToken: string(out.Tokens.Refresh),
		ExpiresAt:    out.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.refresh.Execute(r.Context(), auth.RefreshToken(req.RefreshToken))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrRefreshInvalid),
			errors.Is(err, auth.ErrRefreshExpired):
			writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  string(pair.Access),
		RefreshToken: string(pair.Refresh),
		ExpiresAt:    pair.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	pub, err := h.tokenSvc.PublicKeyPEM()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.WriteHeader(http.StatusOK)
	w.Write(pubPEM)
}
