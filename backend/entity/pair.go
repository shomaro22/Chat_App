package entity

import (
	"bytes"

	"github.com/google/uuid"
)

func OrderedUserPair(a, b uuid.UUID) (smaller, larger uuid.UUID) {
	if bytes.Compare(a[:], b[:]) < 0 {
		return a, b
	}
	return b, a
}
