package m365

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // TOTP (RFC 6238) mandates HMAC-SHA1
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// totpNow returns the current 6-digit TOTP code for a base32 secret, following
// RFC 6238 (HMAC-SHA1, 30s step). Used to answer the MFA prompt during login.
//
//nolint:mnd // RFC 6238 fixed 30-second time step
func totpNow(secret string) (string, error) {
	key, err := decodeBase32Secret(secret)
	if err != nil {
		return "", err
	}
	counter := uint64(time.Now().Unix()) / 30
	return hotp(key, counter), nil
}

// decodeBase32Secret decodes a padded or unpadded, spaced base32 secret.
//
//nolint:mnd // RFC 4648 base32 groups in 8-char blocks
func decodeBase32Secret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	if pad := len(s) % 8; pad != 0 {
		s += strings.Repeat("=", 8-pad)
	}
	key, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("m365: decode TOTP secret: %w", err)
	}
	return key, nil
}

// hotp computes the 6-digit HOTP value for a key and counter.
//
//nolint:mnd // RFC 4226 dynamic-truncation bit masks and shifts
func hotp(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", code%1_000_000)
}
