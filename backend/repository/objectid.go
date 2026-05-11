package repository

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func newObjectID() (string, error) {
	raw := make([]byte, 12)
	now := uint32(time.Now().Unix())
	raw[0] = byte(now >> 24)
	raw[1] = byte(now >> 16)
	raw[2] = byte(now >> 8)
	raw[3] = byte(now)
	if _, err := rand.Read(raw[4:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
