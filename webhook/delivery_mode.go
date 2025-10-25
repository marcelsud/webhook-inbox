package webhook

import "fmt"

/* DeliveryMode represents how webhooks are delivered to target URLs
 * FIFO ensures ordered delivery with parallelism=1
 * PubSub allows concurrent delivery with parallelism>1
 */
type DeliveryMode int

const (
	FIFO DeliveryMode = iota + 1
	PubSub
)

// String returns the string representation of the delivery mode
func (d DeliveryMode) String() string {
	switch d {
	case FIFO:
		return "fifo"
	case PubSub:
		return "pubsub"
	default:
		return "unknown"
	}
}

// NewDeliveryMode creates a DeliveryMode from a string
func NewDeliveryMode(s string) DeliveryMode {
	switch s {
	case "fifo":
		return FIFO
	case "pubsub":
		return PubSub
	default:
		return FIFO // default to FIFO for safety
	}
}

// Validate checks if the delivery mode is valid
func (d DeliveryMode) Validate() error {
	if d != FIFO && d != PubSub {
		return fmt.Errorf("invalid delivery mode: %d", d)
	}
	return nil
}
