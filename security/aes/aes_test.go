package aes

import (
	"testing"

	"github.com/go-pantheon/fabrica-util/xrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	org     = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	short   = []byte("1")
	long, _ = xrand.RandAlphaNumString(1024 * 1024)
	empty   = []byte("")
	utf8    = []byte("测试中文加密解密")
	special = []byte("!@#$%^&*()_+-=[]{}|;:,.<>?")
)

func TestAESGCMCodec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "normal ascii text",
			input: org,
		},
		{
			name:  "short input",
			input: short,
		},
		{
			name:  "long input",
			input: []byte(long),
		},
		{
			name:  "chinese characters",
			input: utf8,
		},
		{
			name:  "special characters",
			input: special,
		},
	}

	key, err := xrand.RandAlphaNumString(32)
	assert.Nil(t, err)

	// Encrypt
	server, err := NewAESCipher([]byte(key))
	require.Nil(t, err)
	client, err := NewAESCipher([]byte(key))
	require.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encrypted, err := server.Encrypt(tt.input)
			assert.Nil(t, err)

			decrypted, err := client.Decrypt(encrypted)
			assert.Nil(t, err)
			assert.Equal(t, tt.input, decrypted)

			encrypted, err = client.Encrypt(tt.input)
			assert.Nil(t, err)

			decrypted, err = server.Decrypt(encrypted)
			assert.Nil(t, err)
			assert.Equal(t, tt.input, decrypted)
		})
	}
}

func TestAESGCMCodec_AllowEmpty(t *testing.T) {
	t.Parallel()

	key, err := xrand.RandAlphaNumString(32)
	assert.Nil(t, err)

	server, err := NewAESCipher([]byte(key))
	require.Nil(t, err)

	client, err := NewAESCipher([]byte(key))
	require.Nil(t, err)

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "normal input",
			input: org,
		},
		{
			name:  "empty input",
			input: empty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encrypted, err := server.EncryptAllowEmpty(tt.input)
			require.NoError(t, err)

			decrypted, err := client.DecryptAllowEmpty(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decrypted)
		})
	}

	// want error if encrypte is wrong
	wrong, err := server.Encrypt(org)
	require.NoError(t, err)

	wrong[len(wrong)-1] = ^wrong[len(wrong)-1]
	_, err = client.DecryptAllowEmpty(wrong)
	require.Error(t, err)
}

func TestAESGCMDecrypt(t *testing.T) {
	t.Parallel()

	key, err := xrand.RandAlphaNumString(32)
	require.Nil(t, err)

	cipher, err := NewAESCipher([]byte(key))
	require.Nil(t, err)

	encrypted, err := cipher.Encrypt(org)
	require.Nil(t, err)

	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "normal input",
			input:   encrypted,
			want:    org,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   []byte(""),
			wantErr: true,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name:    "less than nonce size input",
			input:   encrypted[:cipher.block.NonceSize()-1],
			wantErr: true,
		},
		{
			name:    "less than encrypted size input",
			input:   encrypted[:len(encrypted)-1],
			wantErr: true,
		},
		{
			name:    "more than nonce size input",
			input:   append(encrypted, byte(1)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decrypted, err := cipher.Decrypt(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, decrypted)
			}
		})
	}

	// test decrypt with different key
	key2, err := xrand.RandAlphaNumString(16)
	require.Nil(t, err)

	cipher2, err := NewAESCipher([]byte(key2))
	require.Nil(t, err)

	_, err = cipher2.Decrypt(encrypted)
	require.Error(t, err)
}

func TestNewAESCipher(t *testing.T) {
	t.Parallel()

	key16, err := xrand.RandAlphaNumString(16)
	require.Nil(t, err)
	key24, err := xrand.RandAlphaNumString(24)
	require.Nil(t, err)
	key32, err := xrand.RandAlphaNumString(32)
	require.Nil(t, err)
	key20, err := xrand.RandAlphaNumString(20)
	require.Nil(t, err)

	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{
			name:    "32 bytes key",
			key:     []byte(key32),
			wantErr: false,
		},
		{
			name:    "24 bytes key",
			key:     []byte(key24),
			wantErr: false,
		},
		{
			name:    "16 bytes key",
			key:     []byte(key16),
			wantErr: false,
		},
		{
			name:    "20 bytes key",
			key:     []byte(key20),
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     []byte(""),
			wantErr: true,
		},
		{
			name:    "nil key",
			key:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewAESCipher(tt.key)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func BenchmarkAESGCMEncrypt(b *testing.B) {
	data, err := xrand.RandAlphaNumString(32)
	require.Nil(b, err)

	cipher, err := NewAESCipher([]byte(data))
	require.Nil(b, err)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := cipher.Encrypt(org); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkAESGCMDecrypt(b *testing.B) {
	data, err := xrand.RandAlphaNumString(32)
	require.Nil(b, err)

	cipher, err := NewAESCipher([]byte(data))
	require.Nil(b, err)

	ser, err := cipher.Encrypt(org)
	require.Nil(b, err)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := cipher.Decrypt(ser); err != nil {
				b.Fatal(err)
			}
		}
	})
}
