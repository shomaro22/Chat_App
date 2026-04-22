package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// OpaqueTokenHex はメール用ワンタイムトークン（平文）を生成する（64 hex 文字）。
func OpaqueTokenHex() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashOpaqueToken は DB 保存用の SHA256 ハッシュ（hex）。
func HashOpaqueToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
