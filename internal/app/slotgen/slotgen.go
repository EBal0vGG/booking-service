package slotgen

import "context"

// SlotGenerator defines the slot generation contract.
type SlotGenerator interface {
	Generate(ctx context.Context) error
}
