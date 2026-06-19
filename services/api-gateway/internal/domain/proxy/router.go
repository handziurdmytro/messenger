package proxy

import "fmt"

type Router struct {
	routes []Route
}

func NewRouter(routes []Route) *Router {
	return &Router{routes: routes}
}

func (r *Router) Route(req Request) (Decision, error) {
	route, ok := r.findRoute(req.Path)
	if !ok {
		return Decision{}, DecisionError{
			Code:   ErrRouteNotFound,
			Reason: fmt.Sprintf("no route matched path %q", req.Path),
		}
	}

	if err := r.authorize(route, req); err != nil {
		return Decision{}, err
	}

	return r.buildDecision(route, req), nil
}

func (r *Router) findRoute(path string) (Route, bool) {
	var matched *Route
	for i, route := range r.routes {
		if !route.Match(path) {
			continue
		}
		if matched == nil || len(route.Prefix) > len(matched.Prefix) {
			matched = &r.routes[i]
		}
	}

	if matched == nil {
		return Route{}, false
	}

	return *matched, true
}

func (r *Router) authorize(route Route, req Request) error {
	if !route.AuthRequired {
		return nil
	}

	if req.Identity == nil {
		return DecisionError{
			Code:   ErrAuthRequired,
			Reason: "authentication required",
		}
	}

	return nil
}

func (r *Router) buildDecision(route Route, req Request) Decision {
	decision := Decision{
		TargetURL:      route.Target,
		RewritePath:    route.RewritePath(req.Path),
		ForwardHeaders: make(map[string]string),
	}

	if req.Identity != nil {
		decision.ForwardHeaders["X-User-ID"] = req.Identity.UserID
	}

	return decision
}
