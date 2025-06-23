// Package xid provides utilities for ID generation, encoding, and zone-based ID management
package xid

import (
	"math"
	"strconv"
	"strings"

	"github.com/go-pantheon/fabrica-util/errors"
	"github.com/speps/go-hashids/v2"
)

const (
	uidHashLen = 18
	salt       = "go-pahtheon#2020"
)

const (
	gameIDSlotBit = 16
	gameIDBit     = 63 - gameIDSlotBit
	gameIDMask    = (1 << gameIDBit) - 1
	MaxGameID     = int64(gameIDMask)
	MinGameID     = -MaxGameID
)

const (
	MaxUID = int64(math.MaxInt64)
)

const (
	zoneBit     = 8
	zoneSlotBit = 8
	zoneMask    = (1 << zoneBit) - 1
	// MaxZone is the maximum zone value (255) used in ID encoding
	MaxZone = uint8(zoneMask)
)

var (
	ErrGameIDTooLarge = errors.New("gameID is too large")
	ErrGameIDTooSmall = errors.New("gameID is too small")
)

var (
	h *hashids.HashID
)

func init() {
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = uidHashLen

	var err error
	if h, err = hashids.NewWithData(hd); err != nil {
		panic(errors.Wrapf(err, "hashID encode failed"))
	}
}

// BuildUID combines a zoneID with a zone value to create a combined ID
func BuildUID(gameID int64, zone uint8) (int64, error) {
	if gameID > MaxGameID {
		return 0, ErrGameIDTooLarge
	}

	if gameID < MinGameID {
		return 0, ErrGameIDTooSmall
	}

	return (gameID << gameIDSlotBit) | int64(zone)<<zoneSlotBit, nil
}

// SplitUID splits a combined ID into its zoneID and zone components
func SplitUID(uid int64) (gameID int64, zone uint8) {
	gameID = uid >> gameIDSlotBit

	zoneBits := uid >> zoneSlotBit & zoneMask
	zone = uint8(zoneBits)

	return gameID, zone
}

// EncodeID encodes an ID into a string representation
// Returns the string ID or an error if encoding fails
func EncodeID(id int64) (string, error) {
	if id < 0 {
		return strconv.FormatInt(id, 10), nil
	}

	str, err := h.EncodeInt64([]int64{id})
	if err != nil {
		return "", errors.Wrapf(err, "HashID encode failed. id:%d", id)
	}

	return str, nil
}

// DecodeID decodes a string representation back into an ID
// Returns the decoded ID or an error if decoding fails
func DecodeID(str string) (int64, error) {
	if strings.IndexRune(str, '-') == 0 {
		return strconv.ParseInt(str, 10, 64)
	}

	ids, err := h.DecodeInt64WithError(str)
	if err != nil {
		return 0, errors.Wrapf(err, "HashID decode failed. str:%s", str)
	}

	if len(ids) == 0 {
		return 0, errors.Errorf("HashID decode failed. str:%s", str)
	}

	return ids[0], nil
}
