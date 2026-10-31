package importer

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const maxCertificatesPerFile = 100

type Certificate struct {
	Subject     string
	Issuer      string
	NotAfter    time.Time
	Fingerprint string
}

func ParseCertificatesPEM(data []byte, limits Limits) ([]Certificate, error) {
	if int64(len(data)) > limits.MaxBytes {
		return nil, tooLargeError(limits)
	}
	var certs []Certificate
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		if len(certs) >= maxCertificatesPerFile {
			return nil, fmt.Errorf("%w: more than %d certificates in one file", domain.ErrValidation, maxCertificatesPerFile)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: certificate %d: %v", domain.ErrValidation, len(certs)+1, err)
		}
		sum := sha256.Sum256(cert.Raw)
		certs = append(certs, Certificate{
			Subject:     cert.Subject.String(),
			Issuer:      cert.Issuer.String(),
			NotAfter:    cert.NotAfter.UTC(),
			Fingerprint: hex.EncodeToString(sum[:]),
		})
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("%w: no PEM-encoded CERTIFICATE blocks found", domain.ErrValidation)
	}
	return certs, nil
}
