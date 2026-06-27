// Package id generates 15-char PocketBase-style string PKs. It is a leaf
// package (only crypto/rand) so ent/schema can import it while internal/util
// is free to import ent (GetUser/GetActorID) without an import cycle.
package id

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
