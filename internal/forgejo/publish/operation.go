package publish

import "strconv"

// StatusError reports a bounded native dispatch status with its bounded
// body. Callers map terminal statuses without retrying and reconcile
// after the rest.
type StatusError struct {
	Body   string
	Status int
}

func (e *StatusError) Error() string {
	return "background request returned status " + strconv.Itoa(e.Status)
}
