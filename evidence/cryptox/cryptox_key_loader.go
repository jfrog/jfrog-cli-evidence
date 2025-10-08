package cryptox

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"strings"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/secure-systems-lab/go-securesystemslib/cjson"
)

var KeyIDHashAlgorithms = []string{"sha256", "sha512"}

var (
	ErrNotPrivateKey               = errors.New("loaded key is not a private key")
	ErrSignatureVerificationFailed = errors.New("failed to verify signature")
	ErrUnknownKeyType              = errors.New("unknown key type")
	ErrInvalidThreshold            = errors.New("threshold is either less than 1 or greater than number of provided public keys")
	ErrInvalidKey                  = errors.New("key object has no value")
	ErrInvalidPEM                  = errors.New("unable to parse PEM block")
)

const (
	PublicKeyPEM = "PUBLIC KEY"
)

type SSLibKey struct {
	KeyIDHashAlgorithms []string `json:"keyid_hash_algorithms"`
	KeyType             string   `json:"keytype"`
	KeyVal              KeyVal   `json:"keyval"`
	Scheme              string   `json:"scheme"`
	KeyID               string   `json:"keyid"`
}

type KeyVal struct {
	Private     string `json:"private,omitempty"`
	Public      string `json:"public,omitempty"`
	Certificate string `json:"certificate,omitempty"`
	Identity    string `json:"identity,omitempty"`
	Issuer      string `json:"issuer,omitempty"`
}

func LoadKey(fileContent []byte) (*SSLibKey, error) {
	// Decode PEM block
	pemBlock, key, err := decodeAndParsePEM(fileContent)
	if err != nil {
		return nil, err
	}

	// Create SSLibKey based on key type
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		return createSSLibKeyFromECDSA(k, pemBlock, fileContent)
	case *ecdsa.PublicKey:
		return createSSLibKeyFromECDSAPublic(k, pemBlock)
	case *rsa.PrivateKey:
		return createSSLibKeyFromRSA(k, pemBlock, fileContent)
	case *rsa.PublicKey:
		return createSSLibKeyFromRSAPublic(k, pemBlock)
	case ed25519.PrivateKey:
		return createSSLibKeyFromED25519(k, pemBlock, fileContent)
	case *ed25519.PrivateKey:
		return createSSLibKeyFromED25519(*k, pemBlock, fileContent)
	case ed25519.PublicKey:
		return createSSLibKeyFromED25519Public(k, pemBlock)
	default:
		return nil, errorutils.CheckErrorf("unsupported key type: %T", key)
	}
}

func createSSLibKeyFromECDSA(key *ecdsa.PrivateKey, _ *pem.Block, fileContent []byte) (*SSLibKey, error) {
	keyID, err := calculateKeyIDFromECDSA(key)
	if err != nil {
		return nil, err
	}

	// Generate public key PEM from the private key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	publicKeyPEM := strings.TrimSpace(string(generatePEMBlock(publicKeyBytes, PublicKeyPEM)))

	return &SSLibKey{
		KeyType:             ECDSAKeyType,
		Scheme:              ECDSAKeyScheme,
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyID:               keyID,
		KeyVal:              KeyVal{Private: string(fileContent), Public: publicKeyPEM},
	}, nil
}

func createSSLibKeyFromECDSAPublic(key *ecdsa.PublicKey, pemBlock *pem.Block) (*SSLibKey, error) {
	keyID, err := calculateKeyIDFromECDSAPublic(key)
	if err != nil {
		return nil, err
	}

	return &SSLibKey{
		KeyType:             ECDSAKeyType,
		Scheme:              ECDSAKeyScheme,
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyID:               keyID,
		KeyVal:              KeyVal{Public: strings.TrimSpace(string(pem.EncodeToMemory(pemBlock)))},
	}, nil
}

func createSSLibKeyFromRSA(key *rsa.PrivateKey, _ *pem.Block, fileContent []byte) (*SSLibKey, error) {
	keyID, err := calculateKeyIDFromRSA(key)
	if err != nil {
		return nil, err
	}

	// Generate public key PEM from the private key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	publicKeyPEM := strings.TrimSpace(string(generatePEMBlock(publicKeyBytes, PublicKeyPEM)))

	return &SSLibKey{
		KeyType:             RSAKeyType,
		Scheme:              RSAKeyScheme,
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyID:               keyID,
		KeyVal:              KeyVal{Private: string(fileContent), Public: publicKeyPEM},
	}, nil
}

func createSSLibKeyFromRSAPublic(key *rsa.PublicKey, pemBlock *pem.Block) (*SSLibKey, error) {
	keyID, err := calculateKeyIDFromRSAPublic(key)
	if err != nil {
		return nil, err
	}

	return &SSLibKey{
		KeyType:             RSAKeyType,
		Scheme:              RSAKeyScheme,
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyID:               keyID,
		KeyVal:              KeyVal{Public: strings.TrimSpace(string(pem.EncodeToMemory(pemBlock)))},
	}, nil
}

func createSSLibKeyFromED25519(key ed25519.PrivateKey, _ *pem.Block, fileContent []byte) (*SSLibKey, error) {
	keyID, err := calculateKeyIDFromED25519(key)
	if err != nil {
		return nil, err
	}

	// Store keys as hex strings for ED25519
	publicKey, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errorutils.CheckErrorf("failed to convert to ed25519 public key")
	}
	publicKeyHex := hex.EncodeToString(publicKey)
	privateKeyHex := hex.EncodeToString(key)

	return &SSLibKey{
		KeyType:             ED25519KeyType,
		Scheme:              ED25519KeyScheme,
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyID:               keyID,
		KeyVal:              KeyVal{Private: privateKeyHex, Public: publicKeyHex},
	}, nil
}

func createSSLibKeyFromED25519Public(key ed25519.PublicKey, _ *pem.Block) (*SSLibKey, error) {
	keyID, err := calculateKeyIDFromED25519Public(key)
	if err != nil {
		return nil, err
	}

	// Store public key as hex string for ED25519
	publicKeyHex := hex.EncodeToString(key)

	return &SSLibKey{
		KeyType:             ED25519KeyType,
		Scheme:              ED25519KeyScheme,
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyID:               keyID,
		KeyVal:              KeyVal{Public: publicKeyHex},
	}, nil
}

func calculateKeyIDFromECDSA(key *ecdsa.PrivateKey) (string, error) {
	return calculateKeyIDFromECDSAPublic(&key.PublicKey)
}

func calculateKeyIDFromECDSAPublic(key *ecdsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	return calculateKeyIDFromBytes(publicKeyBytes, ECDSAKeyType, ECDSAKeyScheme)
}

func calculateKeyIDFromRSA(key *rsa.PrivateKey) (string, error) {
	return calculateKeyIDFromRSAPublic(&key.PublicKey)
}

func calculateKeyIDFromRSAPublic(key *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	return calculateKeyIDFromBytes(publicKeyBytes, RSAKeyType, RSAKeyScheme)
}

func calculateKeyIDFromED25519(key ed25519.PrivateKey) (string, error) {
	publicKey, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return "", errorutils.CheckErrorf("failed to convert to ed25519 public key")
	}
	return calculateKeyIDFromED25519Public(publicKey)
}

func calculateKeyIDFromED25519Public(key ed25519.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	return calculateKeyIDFromBytes(publicKeyBytes, ED25519KeyType, ED25519KeyScheme)
}

func calculateKeyIDFromBytes(publicKeyBytes []byte, keyType, scheme string) (string, error) {
	key := map[string]any{
		"keytype":               keyType,
		"scheme":                scheme,
		"keyid_hash_algorithms": KeyIDHashAlgorithms,
		"keyval": map[string]string{
			"public": string(generatePEMBlock(publicKeyBytes, PublicKeyPEM)),
		},
	}

	canonical, err := cjson.EncodeCanonical(key)
	if err != nil {
		return "", errorutils.CheckError(err)
	}

	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}
