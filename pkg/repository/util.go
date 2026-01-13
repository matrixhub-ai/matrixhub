package repository

import (
	"encoding/hex"
)

// isValidSHA checks if a string is a valid 40-character hexadecimal git SHA
func isValidSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
