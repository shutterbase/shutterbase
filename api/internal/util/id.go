package util

import "crypto/rand"

const idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789" // PB id alphabet

func NewID() string {
	b := make([]byte, 15)
	rand.Read(b)
	for i := range b {
		b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
	}
	return string(b)
}

// ponytail: modulo bias over 36 chars is negligible for a 36^15 keyspace; switch to
// rejection sampling only if a collision audit ever flags it.
