package xid

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	mathrand "math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodecID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id int64
	}{
		{id: int64(0)},
		{id: int64(1)},
		{id: int64(2)},
		{id: int64(3)},
		{id: int64(65534)},
		{id: int64(65535)},
		{id: int64(65536)},
		{id: math.MaxInt64},
		{id: math.MaxInt64 - 1},
		{id: int64(-1)},
		{id: -math.MaxInt64},
		{id: -(math.MaxInt64 - 1)},
	}

	for _, tt := range tests {
		str, _ := EncodeID(tt.id)
		id2, _ := DecodeID(str)
		assert.Equal(t, tt.id, id2)
	}
}

// check 5 millions users' id encode str is unique
// func TestIDUnique(t *testing.T) {
// 	v := make(map[string]struct{}, math.MaxInt64)
// 	for id := int64(0); id < 5_000_000; id++ {
// 		str, _ := EncodeID(id)
// 		_, ok := v[str]
// 		assert.False(t, ok)
// 		id2, _ := DecodeID(str)
// 		assert.Equal(t, id, id2)
// 	}
// }

func TestBuildUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		gameID  int64
		zone    uint8
		wantUID int64
		wantErr error
	}{
		{
			name:    "zero values",
			gameID:  0,
			zone:    0,
			wantUID: 0,
			wantErr: nil,
		},
		{
			name:    "small values",
			gameID:  1,
			zone:    2,
			wantUID: (1 << gameIDSlotBit) | (2 << zoneSlotBit),
			wantErr: nil,
		},
		{
			name:    "max zone value",
			gameID:  100,
			zone:    MaxZone,
			wantUID: (100 << gameIDSlotBit) | (int64(MaxZone) << zoneSlotBit),
			wantErr: nil,
		},
		{
			name:    "large gameID up to MaxGameID",
			gameID:  MaxGameID,
			zone:    123,
			wantUID: (MaxGameID << gameIDSlotBit) | (123 << zoneSlotBit),
			wantErr: nil,
		},
		{
			name:    "max values for gameID and zone",
			gameID:  MaxGameID,
			zone:    MaxZone,
			wantUID: (MaxGameID << gameIDSlotBit) | (int64(MaxZone) << zoneSlotBit),
			wantErr: nil,
		},
		{
			name:    "negative gameID",
			gameID:  -1,
			zone:    10,
			wantUID: (-1 << gameIDSlotBit) | (10 << zoneSlotBit),
			wantErr: nil,
		},
		{
			name:    "another negative gameID",
			gameID:  -12345,
			zone:    MaxZone,
			wantUID: (-12345 << gameIDSlotBit) | (int64(MaxZone) << zoneSlotBit),
			wantErr: nil,
		},
		{
			name:    "gameID overflows positive int64 when shifted",
			gameID:  MaxGameID + 1,
			zone:    0,
			wantUID: 0,
			wantErr: ErrGameIDTooLarge,
		},
		{
			name:    "gameID is math.MaxInt64",
			gameID:  math.MaxInt64,
			zone:    0,
			wantUID: 0,
			wantErr: ErrGameIDTooLarge,
		},
		{
			name:    "gameID is MinGameID",
			gameID:  MinGameID - 1,
			zone:    0,
			wantUID: 0,
			wantErr: ErrGameIDTooSmall,
		},
		{
			name:    "gameID is math.MinInt64",
			gameID:  math.MinInt64,
			zone:    0,
			wantUID: 0,
			wantErr: ErrGameIDTooSmall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := BuildUID(tt.gameID, tt.zone)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUID, got, "BuildUID() = %v, want %v", got, tt.wantUID)
			}
		})
	}
}

func TestBuildUIDError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		gameID  int64
		zone    uint8
		wantErr error
	}{
		{
			name:    "gameID too large",
			gameID:  MaxGameID + 1,
			zone:    0,
			wantErr: ErrGameIDTooLarge,
		},
		{
			name:    "gameID too small",
			gameID:  MinGameID - 1,
			zone:    0,
			wantErr: ErrGameIDTooSmall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := BuildUID(tt.gameID, tt.zone)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSplitUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		uid        int64
		wantGameID int64
		wantZone   uint8
	}{
		{
			name:       "zero value",
			uid:        0,
			wantGameID: 0,
			wantZone:   0,
		},
		{
			name:       "small value built by BuildUID",
			uid:        (1 << gameIDSlotBit) | (2 << zoneSlotBit),
			wantGameID: 1,
			wantZone:   2,
		},
		{
			name:       "max zone value built by BuildUID",
			uid:        (100 << gameIDSlotBit) | int64(MaxZone)<<zoneSlotBit,
			wantGameID: 100,
			wantZone:   MaxZone,
		},
		{
			name:       "uid from large gameID",
			uid:        (MaxGameID << gameIDSlotBit) | (123 << zoneSlotBit),
			wantGameID: MaxGameID,
			wantZone:   123,
		},
		{
			name:       "uid from negative gameID",
			uid:        (-12345 << gameIDSlotBit) | (int64(10) << zoneSlotBit),
			wantGameID: -12345,
			wantZone:   10,
		},
		{
			name:       "uid is MaxUID (math.MaxInt64)",
			uid:        MaxUID,
			wantGameID: MaxUID >> gameIDSlotBit,
			wantZone:   uint8((MaxUID >> zoneSlotBit) & zoneMask),
		},
		{
			name:       "uid is -1",
			uid:        -1,
			wantGameID: -1,
			wantZone:   MaxZone,
		},
		{
			name:       "uid is math.MinInt64",
			uid:        math.MinInt64,
			wantGameID: math.MinInt64 >> gameIDSlotBit,
			wantZone:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gameID, zone := SplitUID(tt.uid)
			assert.Equal(t, tt.wantGameID, gameID, "SplitUID() gotGameID = %v, want %v", gameID, tt.wantGameID)
			assert.Equal(t, tt.wantZone, zone, "SplitUID() gotZone = %v, want %v", zone, tt.wantZone)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		gameID int64
		zone   uint8
	}{
		{name: "zero values", gameID: 0, zone: 0},
		{name: "small values", gameID: 1, zone: 2},
		{name: "max zone value", gameID: 100, zone: MaxZone},
		{name: "large gameID", gameID: MaxGameID, zone: 123},
		{name: "max values", gameID: MaxGameID, zone: MaxZone},
		{name: "negative gameID", gameID: -1, zone: 10},
		{name: "another negative gameID", gameID: -12345, zone: MaxZone},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			combined, err := BuildUID(tc.gameID, tc.zone)
			assert.NoError(t, err)

			gotGameID, gotZone := SplitUID(combined)

			assert.Equal(t, tc.gameID, gotGameID, "RoundTrip gameID = %v, want %v", gotGameID, tc.gameID)
			assert.Equal(t, tc.zone, gotZone, "RoundTrip zone = %v, want %v", gotZone, tc.zone)
		})
	}
}

func newRand() *mathrand.Rand {
	// Use crypto/rand for secure seed generation
	var seed1, seed2 uint64

	seedBytes := make([]byte, 16)
	_, err := rand.Read(seedBytes)

	if err != nil {
		// Fallback to time-based seeds with bit masking
		seed1 = uint64(time.Now().UnixMicro()) & 0x7FFFFFFFFFFFFFFF //nolint:gosec // acceptable for tests
		seed2 = uint64(time.Now().UnixMilli()) & 0x7FFFFFFFFFFFFFFF //nolint:gosec // acceptable for tests
	} else {
		seed1 = binary.BigEndian.Uint64(seedBytes[:8])
		seed2 = binary.BigEndian.Uint64(seedBytes[8:])
	}

	return mathrand.New(mathrand.NewPCG(seed1, seed2)) //nolint:gosec // test code with secure seed
}

func BenchmarkEncodeID(b *testing.B) {
	id := newRand().Int64N(math.MaxInt64)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = EncodeID(id)
		}
	})
}

func BenchmarkDecodeID(b *testing.B) {
	id := newRand().Int64N(65535)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			str, _ := EncodeID(id)
			_, _ = DecodeID(str)
		}
	})
}
