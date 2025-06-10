package dht

import (
	"encoding/binary"
	"fmt"
	"time"

	"golang.org/x/crypto/blake2b"
)

type Key = [64]byte

type Value struct {
	Hash    Key
	Expires time.Time
	Payload []byte
}

// Takes a value and serializes it for sharing
func (v Value) Serialize() (serialized []byte) {
	serialized = binary.BigEndian.AppendUint64(serialized, uint64(v.Expires.Unix()))
	serialized = binary.BigEndian.AppendUint64(serialized, uint64(len(v.Payload)))
	serialized = append(serialized, v.Hash[:]...)
	serialized = append(serialized, v.Payload...)
	return serialized
}

// Takes a shared value and deserializes it
func DeserializeValue(serialized []byte) (v Value, err error) {
	v.Expires = time.Unix(int64(binary.BigEndian.Uint64(serialized)), 0)
	payloadSize := binary.BigEndian.Uint64(serialized[8:])
	if n := copy(v.Hash[:], serialized[16:]); n != 64 {
		return v, fmt.Errorf("hash of deserialized value not big enough, got %d", n)
	}
	v.Payload = serialized[16+64:]
	if len(v.Payload) != int(payloadSize) {
		return v, fmt.Errorf("noted payload size and actual length of payload differ: actual length is %d, serialized as %d", len(v.Payload), int(payloadSize))
	}
	if blake2b.Sum512(v.Payload) != v.Hash {
		return v, fmt.Errorf("hash mismatch for deserialized payload!")
	}
	return v, nil
}
