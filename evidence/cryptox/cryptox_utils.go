package cryptox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/secure-systems-lab/go-securesystemslib/cjson"
	"github.com/secure-systems-lab/go-securesystemslib/encrypted"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

/*
Credits: Parts of this file were originally authored for in-toto-golang.
*/

var (
	// ErrNoPEMBlock gets triggered when there is no PEM block in the provided file
	ErrNoPEMBlock = errors.New("failed to decode the data as PEM block (are you sure this is a pem file?)")
	// ErrFailedPEMParsing gets returned when PKCS1, PKCS8 or PKIX key parsing fails
	ErrFailedPEMParsing = errors.New("failed parsing the PEM block: unsupported PEM type")
)

const (
	// EncryptedPrivateKeyPemType PEM types for encrypted private keys
	EncryptedPrivateKeyPemType         = "ENCRYPTED PRIVATE KEY"
	EncryptedCosignPrivateKeyPemType   = "ENCRYPTED COSIGN PRIVATE KEY"
	EncryptedSigstorePrivateKeyPemType = "ENCRYPTED SIGSTORE PRIVATE KEY"
)

func calculateKeyID(k *SSLibKey) (string, error) {
	key := map[string]any{
		"keytype":               k.KeyType,
		"scheme":                k.Scheme,
		"keyid_hash_algorithms": k.KeyIDHashAlgorithms,
		"keyval": map[string]string{
			"public": k.KeyVal.Public,
		},
	}
	canonical, err := cjson.EncodeCanonical(key)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

/*
generatePEMBlock creates a PEM block from scratch via the keyBytes and the pemType.
If successful it returns a PEM block as []byte slice. This function should always
succeed, if keyBytes is empty the PEM block will have an empty byte block.
Therefore only header and footer will exist.
*/
func generatePEMBlock(keyBytes []byte, pemType string) []byte {
	// construct PEM block
	pemBlock := &pem.Block{
		Type:    pemType,
		Headers: nil,
		Bytes:   keyBytes,
	}
	return pem.EncodeToMemory(pemBlock)
}

/*
decodeAndParsePEM receives potential PEM bytes decodes them via pem.Decode
and pushes them to parseKey. If any error occurs during this process,
the function will return nil and an error (either ErrFailedPEMParsing
or ErrNoPEMBlock). On success it will return the decoded pemData, the
key object interface and nil as error. We need the decoded pemData,
because LoadKey relies on decoded pemData for operating system
interoperability.
*/
func decodeAndParsePEM(pemBytes []byte) (*pem.Block, any, error) {
	// Attempt to decode PEM block
	data, _ := pem.Decode(pemBytes)
	if data == nil {
		return nil, nil, errorutils.CheckError(ErrNoPEMBlock)
	}

	if data.Type == "OPENSSH PRIVATE KEY" {
		key, err := parseSSHKey(pemBytes)
		if err != nil {
			return nil, nil, errorutils.CheckError(ErrNoPEMBlock)
		}
		return data, key, nil
	}

	// Check if it's an encrypted key type
	if data.Type == EncryptedPrivateKeyPemType ||
		data.Type == EncryptedCosignPrivateKeyPemType ||
		data.Type == EncryptedSigstorePrivateKeyPemType {
		// For encrypted keys, we need a password - return an error indicating this
		return nil, nil, errorutils.CheckError(errors.New("encrypted private key requires password"))
	}

	// Try to load private key, if this fails try to load key as public key
	key, err := parsePEMKey(data.Bytes)
	if err != nil {
		return nil, nil, errorutils.CheckError(err)
	}
	return data, key, nil
}

func parseSSHKey(keyBytes []byte) (any, error) {
	key, err := ssh.ParseRawPrivateKey(keyBytes)
	if err != nil {
		return nil, errorutils.CheckError(ErrNoPEMBlock)
	}
	switch k := key.(type) {
	case *ecdsa.PrivateKey, *ed25519.PrivateKey, *rsa.PrivateKey:
		return k, nil
	}
	return nil, errorutils.CheckErrorf("PEM parsing failed %v", ErrFailedPEMParsing)
}

/*
parseKey tries to parse a PEM []byte slice. Using the following standards
in the given order:

  - PKCS8
  - PKCS1
  - PKIX
  - ECDSA
  - SSH

On success it returns the parsed key and nil.
On failure it returns nil and the error ErrFailedPEMParsing
*/
func parsePEMKey(data []byte) (any, error) {
	// Try to parse as unencrypted PKCS8 private key first
	key, err := x509.ParsePKCS8PrivateKey(data)
	if err == nil {
		return key, nil
	}
	// Try to parse as PKCS1 RSA private key
	key, err = x509.ParsePKCS1PrivateKey(data)
	if err == nil {
		return key, nil
	}
	// Try to parse as PKIX public key
	key, err = x509.ParsePKIXPublicKey(data)
	if err == nil {
		return key, nil
	}
	// Try to parse as EC private key
	key, err = x509.ParseECPrivateKey(data)
	if err == nil {
		return key, nil
	}

	key, err = parseSSHKey(data)
	if err == nil {
		return key, nil
	}

	return nil, errorutils.CheckErrorf("PEM parsing failed %v", ErrFailedPEMParsing)
}

func hashBeforeSigning(data []byte, h hash.Hash) []byte {
	h.Write(data)
	return h.Sum(nil)
}

// decryptPrivateKey decrypts encrypted private key bytes using password-based encryption
func decryptPrivateKey(encryptedBytes []byte, password []byte) ([]byte, error) {
	decryptedBytes, err := encrypted.Decrypt(encryptedBytes, password)
	if err != nil {
		// Provide user-friendly error messages for common decryption failures
		if strings.Contains(err.Error(), "cipher: message authentication failed") {
			return nil, fmt.Errorf("invalid password: the password you entered is incorrect")
		}
		if strings.Contains(err.Error(), "authentication failed") {
			return nil, fmt.Errorf("invalid password: the password you entered is incorrect")
		}
		if strings.Contains(err.Error(), "decryption failed") {
			return nil, fmt.Errorf("invalid password: the password you entered is incorrect")
		}
		if strings.Contains(err.Error(), "invalid password") {
			return nil, fmt.Errorf("invalid password: the password you entered is incorrect")
		}
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	return decryptedBytes, nil
}

// LoadPrivateKey loads an encrypted private key and returns a crypto.Signer
func LoadPrivateKey(keyBytes []byte, password []byte) (crypto.Signer, error) {
	// Decode PEM block
	p, _ := pem.Decode(keyBytes)
	if p == nil {
		return nil, errorutils.CheckError(errors.New("invalid PEM block"))
	}

	// Check if it's an encrypted key type
	if p.Type != EncryptedPrivateKeyPemType &&
		p.Type != EncryptedCosignPrivateKeyPemType &&
		p.Type != EncryptedSigstorePrivateKeyPemType {
		return nil, errorutils.CheckError(fmt.Errorf("unsupported PEM type: %s", p.Type))
	}

	// Decrypt the private key using the encrypted bytes from the PEM block
	x509Encoded, err := decryptPrivateKey(p.Bytes, password)
	if err != nil {
		// If it's a password error, return it directly without wrapping
		if strings.Contains(err.Error(), "invalid password") {
			return nil, err
		}
		return nil, errorutils.CheckError(fmt.Errorf("decrypt: %w", err))
	}

	// Parse the decrypted PKCS8 private key
	pk, err := x509.ParsePKCS8PrivateKey(x509Encoded)
	if err != nil {
		return nil, errorutils.CheckError(fmt.Errorf("parsing private key: %w", err))
	}

	// Ensure it's a signer
	signer, ok := pk.(crypto.Signer)
	if !ok {
		return nil, errorutils.CheckError(errors.New("private key is not a signer"))
	}

	return signer, nil
}

func hexDecode(t *testing.T, data string) ([]byte, error) {
	t.Helper()
	b, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// CreateVerifier creates dsse.Verifier(s) from an SSLibKey.
func CreateVerifier(publicKey *SSLibKey) ([]dsse.Verifier, error) {
	var verifiers []dsse.Verifier

	switch publicKey.KeyType {
	case ECDSAKeyType:
		ecdsaSinger, err := NewECDSASignerVerifierFromSSLibKey(publicKey)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, ecdsaSinger)
	case RSAKeyType:
		rsaSinger, err := NewRSAPSSSignerVerifierFromSSLibKey(publicKey)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, rsaSinger)
	case ED25519KeyType:
		ed25519Singer, err := NewED25519SignerVerifierFromSSLibKey(publicKey)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, ed25519Singer)
	default:
		return nil, errors.New("unsupported key type")
	}
	return verifiers, nil
}

func GenerateFingerprint(pub crypto.PublicKey) (string, error) {
	if pub == nil {
		return "", errorutils.CheckError(fmt.Errorf("public key not available"))
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)

	if err != nil {
		return "", errorutils.CheckError(fmt.Errorf("failed to marshal public key: %w", err))
	}

	sum256 := sha256.Sum256(pubBytes)
	return base64.StdEncoding.EncodeToString(sum256[:]), nil
}

// PasswordFunc is a function type for getting passwords
type PasswordFunc func(confirm bool) ([]byte, error)

// GetPasswordFromTerm prompts for password from terminal
func GetPasswordFromTerm(confirm bool) ([]byte, error) {
	_, err := fmt.Fprint(os.Stderr, "Enter password for private key: ")
	if err != nil {
		return nil, fmt.Errorf("failed to write to stderr: %w", err)
	}

	pw1, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to write to stderr: %w", err)
	}
	if !confirm {
		return pw1, nil
	}
	_, err = fmt.Fprint(os.Stderr, "Enter password for private key again: ")
	if err != nil {
		return nil, fmt.Errorf("failed to write to stderr: %w", err)
	}

	confirmPassword, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to write to stderr: %w", err)
	}

	if string(pw1) != string(confirmPassword) {
		return nil, errors.New("passwords do not match")
	}
	return pw1, nil
}

// IsTerminal checks if we're running in a terminal
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// GetPassword handles password input
func GetPassword(confirm bool) ([]byte, error) {
	// Check for environment variable first
	if pw := os.Getenv("JFROG_EVIDENCE_PASSWORD"); pw != "" {
		return []byte(pw), nil
	}

	// Check if we're in a terminal
	if IsTerminal() {
		return GetPasswordFromTerm(confirm)
	}

	// Handle piped input (for automation)
	return nil, errors.New("password required but not in terminal and JFROG_EVIDENCE_PASSWORD not set")
}

// encryptPrivateKey encrypts private key bytes using password-based encryption
func encryptPrivateKey(privateKeyBytes []byte, password []byte) ([]byte, error) {
	encryptedBytes, err := encrypted.Encrypt(privateKeyBytes, password)
	if err != nil {
		return nil, errorutils.CheckError(fmt.Errorf("failed to encrypt private key: %w", err))
	}
	return encryptedBytes, nil
}
