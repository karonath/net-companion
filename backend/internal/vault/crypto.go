// Package vault gère le coffre-fort chiffré (PIN, AES-256-GCM, persistance).
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"golang.org/x/crypto/argon2"
)

const keyLen = 32 // AES-256

// deriveKey dérive une clé AES-256 depuis le PIN et le sel via Argon2id.
func deriveKey(pin string, salt []byte) []byte {
	// Paramètres Argon2id : 1 passe, 64 Mio, 4 threads (usage interactif local).
	return argon2.IDKey([]byte(pin), salt, 1, 64*1024, 4, keyLen)
}

// seal chiffre plaintext en AES-256-GCM et renvoie (nonce, ciphertext).
func seal(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// open déchiffre et authentifie ciphertext ; erreur si la clé/le nonce est faux.
func open(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
