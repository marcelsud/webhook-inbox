package webhook

import "fmt"

/* Status represents the current state of a webhook delivery
 * Follows the lifecycle: Pending -> Delivering -> Delivered/Failed/Retrying
 */
type Status int

const (
	Pending Status = iota + 1
	Delivering
	Delivered
	Failed
	Retrying
)

// String returns the string representation of the status
func (s Status) String() string {
	switch s {
	case Pending:
		return "pending"
	case Delivering:
		return "delivering"
	case Delivered:
		return "delivered"
	case Failed:
		return "failed"
	case Retrying:
		return "retrying"
	default:
		return "unknown"
	}
}

// NewStatus creates a Status from a string
func NewStatus(str string) Status {
	switch str {
	case "pending":
		return Pending
	case "delivering":
		return Delivering
	case "delivered":
		return Delivered
	case "failed":
		return Failed
	case "retrying":
		return Retrying
	default:
		return Pending
	}
}

// Validate checks if the status is valid
func (s Status) Validate() error {
	if s < Pending || s > Retrying {
		return fmt.Errorf("invalid status: %d", s)
	}
	return nil
}

// IsFinal returns true if the status is a terminal state
func (s Status) IsFinal() bool {
	return s == Delivered || s == Failed
}
