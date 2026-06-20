package rest

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/auth"
	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/proxy"
)

type WSProxyHandler struct {
	router        *proxy.Router
	authenticator *auth.Authenticator
}

func NewWSProxyHandler(router *proxy.Router, authenticator *auth.Authenticator) *WSProxyHandler {
	return &WSProxyHandler{
		router:        router,
		authenticator: authenticator,
	}
}

func (h *WSProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isWebSocketRequest(r) {
		writeError(w, http.StatusBadRequest, "expected websocket upgrade")
		return
	}

	req := h.buildRequest(r)

	decision, err := h.router.Route(req)
	if err != nil {
		h.handleRouteError(w, err)
		return
	}

	h.tunnel(w, r, decision)
}

func (h *WSProxyHandler) tunnel(w http.ResponseWriter, r *http.Request, decision proxy.Decision) {
	upstreamAddr := targetToAddr(decision.TargetURL)
	upstreamConn, err := net.Dial("tcp", upstreamAddr)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("upstream unavailable: %v", err))
		return
	}
	defer upstreamConn.Close()

	for k, v := range decision.ForwardHeaders {
		r.Header.Set(k, v)
	}

	if err = r.Write(upstreamConn); err != nil {
		writeError(w, http.StatusBadGateway, "failed to forward upgrade request")
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		writeError(w, http.StatusInternalServerError, "hijacking not supported")
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		slog.Error("ws: hijack failed", "error", err)
		return
	}
	defer clientConn.Close()

	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(upstreamConn, clientConn)
		errc <- err
	}()

	go func() {
		_, err := io.Copy(clientConn, upstreamConn)
		errc <- err
	}()

	if err = <-errc; err != nil {
		slog.Error("ws: tunnel closed", "error", err)
	}
}

func (h *WSProxyHandler) buildRequest(r *http.Request) proxy.Request {
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

func (h *WSProxyHandler) handleRouteError(w http.ResponseWriter, err error) {
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

func isWebSocketRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func targetToAddr(targetURL string) string {
	addr := strings.TrimPrefix(targetURL, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	return addr
}
