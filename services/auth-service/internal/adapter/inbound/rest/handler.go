package rest

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/logger"
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
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("register: invalid request body", logger.KeyError, err)
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
			log.Info("register: email already taken", "email", req.Email)
			writeError(w, http.StatusConflict, "email already taken")
		case errors.Is(err, auth.ErrPasswordTooShort):
			log.Info("register: password too short", "email", req.Email)
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error("register: failed", "email", req.Email, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("register: succeeded", logger.KeyUserID, out.UserID)
	writeJSON(w, http.StatusCreated, tokenResponse{
		AccessToken:  string(out.Tokens.Access),
		RefreshToken: string(out.Tokens.Refresh),
		ExpiresAt:    out.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("login: invalid request body", logger.KeyError, err)
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
			log.Info("login: invalid credentials", "email", req.Email)
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			log.Error("login: failed", "email", req.Email, logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("login: succeeded", logger.KeyUserID, out.UserID)
	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  string(out.Tokens.Access),
		RefreshToken: string(out.Tokens.Refresh),
		ExpiresAt:    out.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("refresh: invalid request body", logger.KeyError, err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.refresh.Execute(r.Context(), auth.RefreshToken(req.RefreshToken))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrRefreshInvalid),
			errors.Is(err, auth.ErrRefreshExpired):
			log.Info("refresh: invalid or expired token")
			writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		default:
			log.Error("refresh: failed", logger.KeyError, err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("refresh: succeeded")
	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  string(pair.Access),
		RefreshToken: string(pair.Refresh),
		ExpiresAt:    pair.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	pub, err := h.tokenSvc.PublicKeyPEM()
	if err != nil {
		log.Error("jwks: failed to get public key", logger.KeyError, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		log.Error("jwks: failed to marshal public key", logger.KeyError, err)
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

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(bearer, "Bearer "); ok {
		return after
	}
	return r.URL.Query().Get("token")
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	// Беремо токен з Authorization заголовка (як ти робив у gateway)
	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	identity, err := h.tokenSvc.Authenticate(auth.Token(token)) // У тебе вже є цей метод
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("X-User-ID", string(identity.UserID))
	w.WriteHeader(http.StatusOK)
}
