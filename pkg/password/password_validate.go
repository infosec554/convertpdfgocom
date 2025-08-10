package password

import (
	"errors"
	"regexp"
)

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("Password must be at least 8 characters long")
	}

	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return errors.New("Password must contain at least one uppercase letter")
	}

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return errors.New("Password must contain at least one lowercase letter")
	}

	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return errors.New("Password must contain at least one number")
	}

	if !regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
		return errors.New("Password must contain at least one special character")
	}

	return nil
}
