package ecdh

import (
	"crypto/rand"

	"github.com/go-pantheon/fabrica-util/errors"
	"golang.org/x/crypto/curve25519"
)

// GenKeyPair generate curve25519 key pair
// return private key and public key 32 bytes array or error
func GenKeyPair() (pri [32]byte, pub [32]byte, err error) {
	_, err = rand.Read(pri[:])
	if err != nil {
		return [32]byte{}, [32]byte{}, errors.Wrap(err, "failed to generate random private key")
	}

	curve25519.ScalarBaseMult(&pub, &pri)

	return pri, pub, nil
}

// KeyToBytes convert public key to bytes slice
func KeyToBytes(key *[32]byte) []byte {
	return key[:]
}

// ParseKey parse bytes slice to curve25519 public key
func ParseKey(b []byte) (key [32]byte, err error) {
	if len(b) != 32 {
		return [32]byte{}, errors.New("invalid public key length")
	}

	copy(key[:], b)

	return key, nil
}

// ComputeSharedKey compute shared secret
func ComputeSharedKey(pri [32]byte, pub [32]byte) ([]byte, error) {
	secret, err := curve25519.X25519(pri[:], pub[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute shared secret")
	}

	return secret, nil
}

func ComputePubKey(pri [32]byte) (pub [32]byte, err error) {
	curve25519.ScalarBaseMult(&pub, &pri)
	return pub, nil
}
