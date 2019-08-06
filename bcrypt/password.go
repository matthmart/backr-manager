package bcrypt

import (
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Password represents a generated password
type Password struct {
	Plain  string
	Hashed string
}

// GeneratePassword generates a new random password
func GeneratePassword() (Password, error) {
	pwd := generateRandomString()
	hashed, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return Password{}, fmt.Errorf("unable to hash password using bcrypt: %v", err)
	}

	return Password{Plain: string(pwd), Hashed: string(hashed)}, nil
}

// CompareHashAndPassword compares a password with a hash
// Returns nil on success
func CompareHashAndPassword(hashedPassword, password string) error {
	hash := []byte(hashedPassword)
	pass := []byte(password)

	return bcrypt.CompareHashAndPassword(hash, pass)
}

// https://yourbasic.org/golang/generate-random-string/
func generateRandomString() []byte {
	rand.Seed(time.Now().UnixNano())
	digits := "0123456789"
	specials := "=+%*!@#$?"
	all := "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		digits + specials
	length := 16
	buf := make([]byte, length)
	buf[0] = digits[rand.Intn(len(digits))]
	buf[1] = specials[rand.Intn(len(specials))]
	for i := 2; i < length; i++ {
		buf[i] = all[rand.Intn(len(all))]
	}
	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})

	return buf
}
