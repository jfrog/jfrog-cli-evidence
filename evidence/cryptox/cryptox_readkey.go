package cryptox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

func ReadKey(fileContent []byte) (*SSLibKey, error) {
	// First try to load as unencrypted key
	slibKey, err := LoadKey(fileContent)
	if err != nil {
		// Check if it's an encrypted key error
		if err.Error() == "encrypted private key requires password" {
			// Try to get password from environment variable
			password := os.Getenv("JFROG_EVIDENCE_PASSWORD")
			if password == "" {
				// Try to get password interactively
				passwordBytes, err := GetPassword(false) // Don't confirm for key loading
				if err != nil {
					return nil, fmt.Errorf("failed to get password for encrypted private key: %w", err)
				}
				password = string(passwordBytes)
			}

			// Load encrypted private key
			signer, err := LoadPrivateKey(fileContent, []byte(password))
			if err != nil {
				// If it's a password error, return it directly without wrapping
				if strings.Contains(err.Error(), "invalid password") {
					return nil, err
				}
				return nil, fmt.Errorf("failed to load encrypted private key: %w", err)
			}

			// Convert signer to SSLibKey format
			return signerToSSLibKey(signer, fileContent)
		}
		return nil, err
	}
	if slibKey.KeyVal.Private != "" {
		return slibKey, nil
	}

	return nil, nil
}

// signerToSSLibKey converts a crypto.Signer to SSLibKey format
func signerToSSLibKey(signer crypto.Signer, originalPEM []byte) (*SSLibKey, error) {
	// Get the public key
	publicKey := signer.Public()

	// Marshal public key to PKIX format
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	// Create PEM block for public key
	pubKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}))

	// Determine key type and scheme
	var keyType, scheme string
	switch signer.(type) {
	case *ecdsa.PrivateKey:
		keyType = ECDSAKeyType
		scheme = ECDSAKeyScheme
	case *rsa.PrivateKey:
		keyType = RSAKeyType
		scheme = RSAKeyScheme
	case ed25519.PrivateKey:
		keyType = ED25519KeyType
		scheme = ED25519KeyType
	default:
		return nil, errorutils.CheckError(errors.New("unsupported key type"))
	}

	// Create SSLibKey with the original encrypted PEM as private key
	return &SSLibKey{
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyType:             keyType,
		KeyVal: KeyVal{
			Public:  strings.TrimSpace(pubKeyPEM),
			Private: strings.TrimSpace(string(originalPEM)), // Keep original encrypted PEM
		},
		Scheme: scheme,
	}, nil
}

func ReadPublicKey(fileContent []byte) (*SSLibKey, error) {
	slibKey, err := LoadKey(fileContent)
	if err != nil {
		return nil, err
	}
	if slibKey.KeyVal.Public != "" {
		return slibKey, nil
	}

	return nil, nil
}
