package postgres

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
)

var randomReader io.Reader = rand.Reader

func newID(prefix string) (string, error) {
	clean := strings.TrimSpace(prefix)
	if clean == "" {
		clean = "id"
	}
	var buf [8]byte
	if _, err := io.ReadFull(randomReader, buf[:]); err != nil {
		return "", fmt.Errorf("generate random id bytes: %w", err)
	}
	return fmt.Sprintf("%s_%d_%s", clean, time.Now().UTC().UnixNano(), hex.EncodeToString(buf[:])), nil
}
