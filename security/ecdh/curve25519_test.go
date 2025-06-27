package ecdh

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/go-pantheon/fabrica-util/security/aes"
	"github.com/stretchr/testify/assert"
)

func TestKeyExchange(t *testing.T) {
	t.Parallel()

	svrPri, svrPub, err := GenKeyPair()
	assert.NoError(t, err)

	cliPri, cliPub, err := GenKeyPair()
	assert.NoError(t, err)

	svrSharedKey, err := ComputeSharedKey(svrPri, cliPub)
	assert.NoError(t, err)

	cliSharedKey, err := ComputeSharedKey(cliPri, svrPub)
	assert.NoError(t, err)

	assert.Equal(t, svrSharedKey, cliSharedKey)
}

func TestInvalidPubKey(t *testing.T) {
	t.Parallel()

	invalidKey := make([]byte, 31)

	_, err := rand.Read(invalidKey)
	assert.NoError(t, err)

	_, err = ParseKey(invalidKey)
	assert.Error(t, err)
}

func TestAllZeroPubKey(t *testing.T) {
	t.Parallel()

	// Generate a valid private key
	pri, _, err := GenKeyPair()
	assert.NoError(t, err)

	// Create an all-zero public key (invalid)
	var zeroPub [32]byte

	// This should fail because all-zero public key is a low order point
	_, err = ComputeSharedKey(pri, zeroPub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "low order point")
}

func TestSharedSecret(t *testing.T) {
	t.Parallel()

	for i := range 100 {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			t.Parallel()

			svrPri, _, err := GenKeyPair()
			assert.NoError(t, err)

			_, cliPub, err := GenKeyPair()
			assert.NoError(t, err)

			secret, err := ComputeSharedKey(svrPri, cliPub)
			assert.NoError(t, err)

			cipher, err := aes.NewAESCipher(secret)
			assert.NoError(t, err)

			plaintext := []byte("Hello, world!")

			encrypted, err := cipher.Encrypt(plaintext)
			assert.NoError(t, err)

			decrypted, err := cipher.Decrypt(encrypted)
			assert.NoError(t, err)
			assert.Equal(t, plaintext, decrypted)
		})
	}
}

func TestComputePubKey(t *testing.T) {
	t.Parallel()

	pri, pub, err := GenKeyPair()
	assert.NoError(t, err)

	pub2, err := ComputePubKey(pri)
	assert.NoError(t, err)

	assert.Equal(t, pub, pub2)
}

func BenchmarkKeyGeneration(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := GenKeyPair()
			if err != nil {
				b.Fatalf("key generation failed: %v", err)
			}
		}
	})
}

func BenchmarkSharedSecretComputation(b *testing.B) {
	// pre-generate fixed key pair
	svrPri, svrPub, _ := GenKeyPair()
	cliPri, cliPub, _ := GenKeyPair()

	// reset timer
	b.ResetTimer()
	b.ReportAllocs()

	// parallel test
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// test two direction key computation
			if _, err := ComputeSharedKey(svrPri, cliPub); err != nil {
				b.Fatal(err)
			}

			if _, err := ComputeSharedKey(cliPri, svrPub); err != nil {
				b.Fatal(err)
			}
		}
	})
}

var sink any // prevent compiler optimize

func BenchmarkKeyGenerationAndExchange(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// generate server key
		svrPri, _, err := GenKeyPair()
		if err != nil {
			b.Fatal(err)
		}

		// generate client key
		_, cliPub, err := GenKeyPair()
		if err != nil {
			b.Fatal(err)
		}

		// compute shared secret
		secret, err := ComputeSharedKey(svrPri, cliPub)
		if err != nil {
			b.Fatal(err)
		}

		// prevent compiler optimize
		sink = secret
	}
}
