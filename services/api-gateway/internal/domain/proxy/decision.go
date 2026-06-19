package proxy

type DecisionErrorCode int

const (
	ErrRouteNotFound DecisionErrorCode = iota
	ErrAuthRequired
	ErrForbidden
)

type DecisionError struct {
	Code   DecisionErrorCode
	Reason string
}

type Decision struct {
	TargetURL      string
	RewritePath    string
	ForwardHeaders map[string]string
}

func (e DecisionError) Error() string {
	return e.Reason
}
