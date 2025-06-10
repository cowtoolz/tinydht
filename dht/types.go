package dht

import "time"

type Key = [64]byte

type Value struct {
	Hash    Key
	Expires time.Time
	Payload []byte
}

type Command = uint8

const (
	// HELLO makes this peer known to other peers, requesting that they send their peer list and their known keys
	HELLO Command = iota

	// GOODBYE tells peers to remove this peer from their peer list
	GOODBYE

	// PEERS announces to other peers the full list of known peers
	PEERS

	// KEYS announces to other peers that the value for a key is known
	KEYS

	// VALUE is a value sent over the wire to place into the local table
	VALUE = 255
)
