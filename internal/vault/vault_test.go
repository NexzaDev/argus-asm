package vault

import (
	"bytes"
	"errors"
	"testing"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const testIterations = minIterations

func TestSealOpenRoundTrip(t *testing.T) {
	plain := []byte("SQLite format 3\x00 synthetic payload")
	sealed, err := Seal(plain, []byte("correct horse"), testIterations)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if bytes.Contains(sealed, []byte("synthetic payload")) {
		t.Fatal("sealed output contains the plaintext")
	}
	got, err := Open(sealed, []byte("correct horse"))
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("Open() = %q, %v", got, err)
	}
}

func TestSealUsesFreshSaltAndNonce(t *testing.T) {
	a, _ := Seal([]byte("same"), []byte("correct horse battery"), testIterations)
	b, _ := Seal([]byte("same"), []byte("correct horse battery"), testIterations)
	if bytes.Equal(a, b) {
		t.Fatal("two seals of the same input are identical")
	}
}

func TestOpenRejectsWrongPassphraseAndDamage(t *testing.T) {
	sealed, err := Seal([]byte("payload"), []byte("correct horse battery"), testIterations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(sealed, []byte("wrong")); !errors.Is(err, ErrWrongPassphraseOrDamaged) {
		t.Fatalf("wrong passphrase: err = %v", err)
	}
	for name, mutate := range map[string]func([]byte) []byte{
		"header iteration byte": func(d []byte) []byte { d[len(magic)+3] ^= 1; return d },
		"salt byte":             func(d []byte) []byte { d[len(magic)+5] ^= 1; return d },
		"nonce byte":            func(d []byte) []byte { d[headerLength-1] ^= 1; return d },
		"ciphertext byte":       func(d []byte) []byte { d[len(d)-1] ^= 1; return d },
		"truncated":             func(d []byte) []byte { return d[:len(d)-1] },
	} {
		copyData := append([]byte(nil), sealed...)
		if _, err := Open(mutate(copyData), []byte("correct horse battery")); err == nil {
			t.Fatalf("%s: damaged file was accepted", name)
		}
	}
}

func TestOpenRejectsForeignAndShortInput(t *testing.T) {
	if _, err := Open([]byte("SQLite format 3\x00 not sealed at all, long enough to pass the length check........"), []byte("correct horse battery")); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("foreign input: err = %v", err)
	}
	if _, err := Open([]byte("ARGUSDB1"), []byte("correct horse battery")); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("short input: err = %v", err)
	}
}

func TestSealValidatesArguments(t *testing.T) {
	if _, err := Seal([]byte("x"), []byte("short"), testIterations); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("short passphrase: err = %v", err)
	}
	if _, err := Seal([]byte("x"), nil, testIterations); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty passphrase: err = %v", err)
	}
	if _, err := Seal([]byte("x"), []byte("correct horse battery"), minIterations-1); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("low iterations: err = %v", err)
	}
	if _, err := Seal([]byte("x"), []byte("correct horse battery"), maxIterations+1); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("high iterations: err = %v", err)
	}
}
