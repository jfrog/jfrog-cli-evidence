package cryptox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

const (
	ECDSAKeyType   = "ecdsa"
	ECDSAKeyScheme = "ecdsa-sha2-nistp256"
)

// ECDSASignerVerifier is a dsse.SignerVerifier compliant interface to sign and
// verify signatures using ECDSA keys.
type ECDSASignerVerifier struct {
	keyID     string
	curveSize int
	private   *ecdsa.PrivateKey
	public    *ecdsa.PublicKey
}

// NewECDSASignerVerifierFromSSLibKey creates an ECDSASignerVerifier from an
// SSLibKey.
func NewECDSASignerVerifierFromSSLibKey(key *SSLibKey) (*ECDSASignerVerifier, error) {
	if len(key.KeyVal.Public) == 0 {
		return nil, errorutils.CheckError(ErrInvalidKey)
	}

	_, publicParsedKey, err := decodeAndParsePEM([]byte(key.KeyVal.Public))
	if err != nil {
		return nil, errorutils.CheckError(fmt.Errorf("unable to create ECDSA signerverifier: %w", err))
	}

	puk, ok := publicParsedKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errorutils.CheckError(fmt.Errorf("couldnt convert to ecdsa public key"))
	}
	sv := &ECDSASignerVerifier{
		keyID:     key.KeyID,
		curveSize: puk.Params().BitSize,
		public:    puk,
		private:   nil,
	}

	if len(key.KeyVal.Private) > 0 {
		// Check if it's an encrypted key
		if strings.Contains(key.KeyVal.Private, "ENCRYPTED") {
			// Try to get password from environment variable
			password := os.Getenv("JFROG_EVIDENCE_PASSWORD")
			if password == "" {
				// Try to get password interactively
				passwordBytes, err := GetPassword(false) // Don't confirm for signing
				if err != nil {
					return nil, errorutils.CheckError(fmt.Errorf("unable to get password for encrypted key: %w", err))
				}
				password = string(passwordBytes)
			}

			// Load encrypted private key
			signer, err := LoadPrivateKey([]byte(key.KeyVal.Private), []byte(password))
			if err != nil {
				// If it's a password error, return it directly without wrapping
				if strings.Contains(err.Error(), "invalid password") {
					return nil, err
				}
				return nil, errorutils.CheckError(fmt.Errorf("unable to load encrypted private key: %w", err))
			}

			pk, ok := signer.(*ecdsa.PrivateKey)
			if !ok {
				return nil, errorutils.CheckError(fmt.Errorf("encrypted key is not an ECDSA private key"))
			}
			sv.private = pk
		} else {
			// Handle unencrypted key as before
			_, privateParsedKey, err := decodeAndParsePEM([]byte(key.KeyVal.Private))
			if err != nil {
				return nil, errorutils.CheckError(fmt.Errorf("unable to create ECDSA signerverifier: %w", err))
			}

			pk, ok := privateParsedKey.(*ecdsa.PrivateKey)
			if !ok {
				return nil, errorutils.CheckError(fmt.Errorf("couldnt convert to ecdsa private key"))
			}
			sv.private = pk
		}
	}

	return sv, nil
}

// Sign creates a signature for `data`.
func (sv *ECDSASignerVerifier) Sign(data []byte) ([]byte, error) {
	if sv.private == nil {
		return nil, errorutils.CheckError(ErrNotPrivateKey)
	}

	hashedData := getECDSAHashedData(data, sv.curveSize)

	return ecdsa.SignASN1(rand.Reader, sv.private, hashedData)
}

// Verify verifies the `sig` value passed in against `data`.
func (sv *ECDSASignerVerifier) Verify(data []byte, sig []byte) error {
	hashedData := getECDSAHashedData(data, sv.curveSize)

	if ok := ecdsa.VerifyASN1(sv.public, hashedData, sig); !ok {
		return errorutils.CheckError(ErrSignatureVerificationFailed)
	}

	return nil
}

// KeyID returns the identifier of the key used to create the
// ECDSASignerVerifier instance.
func (sv *ECDSASignerVerifier) KeyID() (string, error) {
	return sv.keyID, nil
}

// Public returns the public portion of the key used to create the
// ECDSASignerVerifier instance.
func (sv *ECDSASignerVerifier) Public() crypto.PublicKey {
	return sv.public
}

func getECDSAHashedData(data []byte, curveSize int) []byte {
	switch {
	case curveSize <= 256:
		return hashBeforeSigning(data, sha256.New())
	case 256 < curveSize && curveSize <= 384:
		return hashBeforeSigning(data, sha512.New384())
	case curveSize > 384:
		return hashBeforeSigning(data, sha512.New())
	}
	return []byte{}
}

// GenerateECDSAKeyPair generates a new ECDSA P-256 key pair and returns it as PEM-encoded strings
func GenerateECDSAKeyPair() (privateKeyPEM, publicKeyPEM string, err error) {
	return GenerateECDSAKeyPairWithPassword(nil)
}

// GenerateECDSAKeyPairWithPassword generates a new ECDSA P-256 key pair with optional password encryption
func GenerateECDSAKeyPairWithPassword(pf PasswordFunc) (privateKeyPEM, publicKeyPEM string, err error) {
	// Generate ECDSA key pair with P-256 curve
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", errorutils.CheckError(fmt.Errorf("failed to generate ECDSA key pair: %w", err))
	}

	// Marshal private key to PKCS#8 format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", errorutils.CheckError(fmt.Errorf("failed to marshal private key: %w", err))
	}

	// Handle password encryption if provided
	if pf != nil {
		password, err := pf(true) // Always confirm password
		if err != nil {
			return "", "", errorutils.CheckError(fmt.Errorf("failed to get password: %w", err))
		}

		// Encrypt the private key
		encryptedBytes, err := encryptPrivateKey(privateKeyBytes, password)
		if err != nil {
			return "", "", errorutils.CheckError(fmt.Errorf("failed to encrypt private key: %w", err))
		}

		// Create PEM block for encrypted private key
		privateKeyBlock := &pem.Block{
			Type:  "ENCRYPTED PRIVATE KEY",
			Bytes: encryptedBytes,
		}
		privateKeyPEM = string(pem.EncodeToMemory(privateKeyBlock))
	} else {
		// Create PEM block for unencrypted private key
		privateKeyBlock := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyBytes,
		}
		privateKeyPEM = string(pem.EncodeToMemory(privateKeyBlock))
	}

	// Marshal public key to PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", errorutils.CheckError(fmt.Errorf("failed to marshal public key: %w", err))
	}

	// Create PEM block for public key
	publicKeyPEM = string(generatePEMBlock(publicKeyBytes, PublicKeyPEM))

	return privateKeyPEM, publicKeyPEM, nil
}
