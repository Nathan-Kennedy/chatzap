package securestring

import "crypto/subtle"

// Equal compara strings em tempo constante (evita timing attacks).
func Equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
