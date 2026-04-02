package slotgen

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Deterministic slot IDs: same (room, start instant in UTC) always maps to the same UUID.
var slotIDNamespace = uuid.MustParse("c7f8b2e1-4c1a-5f8b-9ad2-9d0b7c3e8f1a")

func stableSlotID(roomID uuid.UUID, startUTC time.Time) uuid.UUID {
	name := roomID.String() + "|" + strconv.FormatInt(startUTC.UTC().UnixNano(), 10)
	return uuid.NewSHA1(slotIDNamespace, []byte(name))
}
