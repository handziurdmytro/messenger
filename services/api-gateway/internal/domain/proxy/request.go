package proxy

type Request struct {
	Method   string
	Path     string
	Headers  map[string][]string
	Identity *AuthIdentity
}

type AuthIdentity struct {
	UserID string
	Roles  []string
}
