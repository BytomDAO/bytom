package sm2

const (
	// PublicKeySize is the size, in bytes, of public keys as used in this package.
	PubKeySize = 33
	// PrivateKeySize is the size, in bytes, of private keys as used in this package.
	PrivKeySize = 32
	// SignatureSize is the size, in bytes, of signatures generated and verified by this package.
	SignatureSize = 64
)

type PubKey []byte
type PrivKey []byte
