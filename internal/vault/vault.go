package vault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const (
	magic        = "ARGUSDB1"
	saltLength   = 16
	nonceLength  = 12
	keyLength    = 32
	headerLength = len(magic) + 4 + saltLength + nonceLength

	DefaultIterations = 600000

	MinPassphraseLength = 12
	minIterations       = 100000
	maxIterations       = 10000000
	MaxPlaintext        = 1 << 30
)

var ErrWrongPassphraseOrDamaged = errors.New("wrong passphrase, or the sealed file is damaged")

func Seal(plaintext []byte, passphrase []byte, iterations int) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("%w: passphrase must not be empty", domain.ErrValidation)
	}
	if len(passphrase) < MinPassphraseLength {
		return nil, fmt.Errorf("%w: passphrase must be at least %d bytes", domain.ErrValidation, MinPassphraseLength)
	}
	if iterations < minIterations || iterations > maxIterations {
		return nil, fmt.Errorf("%w: iterations must be between %d and %d", domain.ErrValidation, minIterations, maxIterations)
	}
	if len(plaintext) > MaxPlaintext {
		return nil, fmt.Errorf("%w: database is larger than %d bytes", domain.ErrValidation, MaxPlaintext)
	}
	header := make([]byte, headerLength)
	copy(header, magic)
	binary.BigEndian.PutUint32(header[len(magic):], uint32(iterations))
	salt := header[len(magic)+4 : len(magic)+4+saltLength]
	nonce := header[len(magic)+4+saltLength:]
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("read random salt: %w", err)
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("read random nonce: %w", err)
	}
	aead, err := newAEAD(passphrase, salt, iterations)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, headerLength+len(plaintext)+aead.Overhead())
	out = append(out, header...)
	return aead.Seal(out, nonce, plaintext, header), nil
}

func Open(data []byte, passphrase []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("%w: passphrase must not be empty", domain.ErrValidation)
	}
	if len(data) < headerLength || !bytes.Equal(data[:len(magic)], []byte(magic)) {
		return nil, fmt.Errorf("%w: not an argus-asm sealed database", domain.ErrValidation)
	}
	header := data[:headerLength]
	iterations := int(binary.BigEndian.Uint32(header[len(magic):]))
	if iterations < minIterations || iterations > maxIterations {
		return nil, fmt.Errorf("%w: sealed file has an unsupported iteration count", domain.ErrValidation)
	}
	salt := header[len(magic)+4 : len(magic)+4+saltLength]
	nonce := header[len(magic)+4+saltLength:]
	aead, err := newAEAD(passphrase, salt, iterations)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, nonce, data[headerLength:], header)
	if err != nil {
		return nil, ErrWrongPassphraseOrDamaged
	}
	return plaintext, nil
}

func newAEAD(passphrase, salt []byte, iterations int) (cipher.AEAD, error) {
	key, err := pbkdf2.Key(sha256.New, string(passphrase), salt, iterations, keyLength)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return aead, nil
}
