package password

import "golang.org/x/crypto/bcrypt"

// Hash generates a bcrypt hash of the given plain string
func Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(bytes), err
}

// Check verifies if the plain string matches the hash
func Check(plain, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}
