package cryptoagent

import "testing"

func TestEncryptDecrypt_roundtrip(t *testing.T) {
	const pass = "uma-passphrase-longa-para-teste-32+"
	plain := "sk-test-key-1234567890"
	cipher, err := Encrypt(plain, pass)
	if err != nil {
		t.Fatal(err)
	}
	if cipher == "" || cipher == plain {
		t.Fatal("ciphertext inesperado")
	}
	out, err := Decrypt(cipher, pass)
	if err != nil {
		t.Fatal(err)
	}
	if out != plain {
		t.Fatalf("got %q want %q", out, plain)
	}
}

func TestDecrypt_wrongPassphrase(t *testing.T) {
	cipher, err := Encrypt("secret", "correct-horse-battery-staple-xyz")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Decrypt(cipher, "wrong-passphrase-xxxxxxxxxxxxxx")
	if err == nil {
		t.Fatal("esperava erro")
	}
}

func TestEncrypt_emptyPassphrase(t *testing.T) {
	_, err := Encrypt("x", "")
	if err == nil {
		t.Fatal("esperava erro")
	}
}
