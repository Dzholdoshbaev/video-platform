package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type IDGenerator func() string

func NewID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		// Entropy failure means the process cannot safely create persistent identities.
		panic(fmt.Sprintf("generate random ID: %v", err))
	}
	return hex.EncodeToString(bytes[:])
}
