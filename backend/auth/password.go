package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword は登録時用に bcrypt でハッシュする。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ComparePassword は保存済みハッシュと平文パスワードを照合する。
func ComparePassword(hashed string, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
