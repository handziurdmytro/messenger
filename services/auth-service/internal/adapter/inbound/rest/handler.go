package rest

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/httpjson"
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

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register godoc
// @Summary      User registration
// @Description  Creates a new user by email and password and returns tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequest true "Registration data"
// @Success      201 {object} tokenResponse
// @Failure      400 {object} errorResponse "Invalid request data or password too short"
// @Failure      409 {object} errorResponse "Email already taken"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req registerRequest
	if err := httpjson.Read(w, r, &req); err != nil {
		log.Warn("register: invalid request body", logger.KeyError, err)
		_ = httpjson.WriteReadError(w, err)
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
			_ = httpjson.Error(w, http.StatusConflict, "email already taken")
		case errors.Is(err, auth.ErrPasswordTooShort):
			log.Info("register: password too short", "email", req.Email)
			_ = httpjson.Error(w, http.StatusBadRequest, err.Error())
		default:
			log.Error("register: failed", "email", req.Email, logger.KeyError, err)
			_ = httpjson.Error(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("register: succeeded", logger.KeyUserID, out.UserID)
	_ = httpjson.Write(w, http.StatusCreated, tokenResponse{
		AccessToken:  string(out.Tokens.Access),
		RefreshToken: string(out.Tokens.Refresh),
		ExpiresAt:    out.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Login godoc
// @Summary      User login
// @Description  Validates credentials and returns a pair of tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequest true "Login data"
// @Success      200 {object} tokenResponse
// @Failure      400 {object} errorResponse "Invalid request data"
// @Failure      401 {object} errorResponse "Invalid email or password"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req registerRequest
	if err := httpjson.Read(w, r, &req); err != nil {
		log.Warn("login: invalid request body", logger.KeyError, err)
		_ = httpjson.WriteReadError(w, err)
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
			_ = httpjson.Error(w, http.StatusUnauthorized, "invalid credentials")
		default:
			log.Error("login: failed", "email", req.Email, logger.KeyError, err)
			_ = httpjson.Error(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("login: succeeded", logger.KeyUserID, out.UserID)
	_ = httpjson.Write(w, http.StatusOK, tokenResponse{
		AccessToken:  string(out.Tokens.Access),
		RefreshToken: string(out.Tokens.Refresh),
		ExpiresAt:    out.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Refresh godoc
// @Summary      Refresh Access token
// @Description  Generates a new pair of tokens based on a valid Refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body refreshRequest true "Refresh token"
// @Success      200 {object} tokenResponse
// @Failure      400 {object} errorResponse "Invalid request data"
// @Failure      401 {object} errorResponse "Invalid or expired token"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	var req refreshRequest
	if err := httpjson.Read(w, r, &req); err != nil {
		log.Warn("refresh: invalid request body", logger.KeyError, err)
		_ = httpjson.WriteReadError(w, err)
		return
	}

	pair, err := h.refresh.Execute(r.Context(), auth.RefreshToken(req.RefreshToken))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrRefreshInvalid),
			errors.Is(err, auth.ErrRefreshExpired):
			log.Info("refresh: invalid or expired token")
			_ = httpjson.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		default:
			log.Error("refresh: failed", logger.KeyError, err)
			_ = httpjson.Error(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	log.Info("refresh: succeeded")
	_ = httpjson.Write(w, http.StatusOK, tokenResponse{
		AccessToken:  string(pair.Access),
		RefreshToken: string(pair.Refresh),
		ExpiresAt:    pair.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

// JWKS godoc
// @Summary      Get public keys (JWKS)
// @Description  Returns the public key in PEM format for external token validation
// @Tags         auth
// @Produce      application/x-pem-file
// @Success      200 {string} string "Public key in PEM format"
// @Failure      500 {object} errorResponse "Internal server error"
// @Router       /.well-known/jwks.pem [get]
func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	log := logger.WithComponent(logger.FromContext(r.Context()), "auth")

	pub, err := h.tokenSvc.PublicKeyPEM()
	if err != nil {
		log.Error("jwks: failed to get public key", logger.KeyError, err)
		_ = httpjson.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		log.Error("jwks: failed to marshal public key", logger.KeyError, err)
		_ = httpjson.Error(w, http.StatusInternalServerError, "internal error")
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

// Validate godoc
// @Summary      Validate Access token
// @Description  Validates the access token and returns status with user ID
// @Tags         auth
// @Security     BearerAuth
// @Param        token query string false "Token (passed if no Bearer header is present)"
// @Success      200 {string} string "Token is valid"
// @Failure      401 {string} string "Missing or invalid token"
// @Header       200 {string} X-User-ID "ID of the validated user"
// @Router       /validate [get]
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	identity, err := h.tokenSvc.Authenticate(auth.Token(token))
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("X-User-ID", string(identity.UserID))
	w.WriteHeader(http.StatusOK)
}
