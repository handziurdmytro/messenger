package rest

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/auth"
	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/proxy"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Handler struct {
	router        *proxy.Router
	authenticator *auth.Authenticator
}

func NewHandler(router *proxy.Router, authenticator *auth.Authenticator) *Handler {
	return &Handler{
		router:        router,
		authenticator: authenticator,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	req := h.buildRequest(r)

	decision, err := h.router.Route(req)
	if err != nil {
		log.Warn("gateway: routing failed", "path", r.URL.Path, "error", err)
		h.handleRouteError(w, err)
		return
	}

	log.Info("gateway: routed", "path", r.URL.Path, "target", decision.TargetURL, "rewrite_path", decision.RewritePath)
	h.proxy(w, r, decision)
}

func (h *Handler) buildRequest(r *http.Request) proxy.Request {
	req := proxy.Request{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: map[string][]string(r.Header),
	}

	if identity := identityFromContext(r.Context()); identity != nil {
		req.Identity = &proxy.AuthIdentity{
			UserID: identity.UserID,
			Roles:  rolesToStrings(identity.Roles),
		}
	}

	return req
}

func (h *Handler) proxy(w http.ResponseWriter, r *http.Request, decision proxy.Decision) {
	log := logger.FromContext(r.Context())

	target, err := url.Parse(decision.TargetURL)
	if err != nil {
		log.Error("gateway: invalid target url", "target", decision.TargetURL, "error", err)
		writeError(w, http.StatusBadGateway, "invalid target")
		return
	}

	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)

			pr.Out.URL.Path = decision.RewritePath

			pr.SetXForwarded()

			for k, v := range decision.ForwardHeaders {
				pr.Out.Header.Set(k, v)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("gateway: upstream error", "target", target.String(), "error", err)
			writeError(w, http.StatusBadGateway, fmt.Sprintf("upstream error: %v", err))
		},
	}

	rp.ServeHTTP(w, r)
}

func (h *Handler) handleRouteError(w http.ResponseWriter, err error) {
	var decErr proxy.DecisionError
	if !errorAs(err, &decErr) {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	switch decErr.Code {
	case proxy.ErrRouteNotFound:
		writeError(w, http.StatusNotFound, decErr.Reason)
	case proxy.ErrAuthRequired:
		writeError(w, http.StatusUnauthorized, decErr.Reason)
	case proxy.ErrForbidden:
		writeError(w, http.StatusForbidden, decErr.Reason)
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func rolesToStrings(roles []auth.Role) []string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return out
}
