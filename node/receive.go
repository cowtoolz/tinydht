package node

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/cowtools/tinydht/dht"
	"golang.org/x/crypto/blake2b"
)

func (c *Client) ReceiveCommand() error {
	errs := []error{}

	var buf [2048]byte
	_, from, err := c.listener.ReadFromUDP(buf[:])
	if err != nil {
		return err
	}

	fmt.Println("RECEIVING:", from.String())

	c.mtx.Lock()
	defer c.mtx.Unlock()

	cmd := buf[0]
	payload := buf[1:]

	switch cmd {
	case dht.HELLO:
		if c.peerKnown(from) {
			errs = append(errs, fmt.Errorf("HELLO recieved from known peer %s", from.String()))
			break
		}
		c.addPeerUnchecked(from)
	case dht.GOODBYE:
		if !c.peerKnown(from) {
			errs = append(errs, fmt.Errorf("GOODBYE recieved from unknown peer %s", from.String()))
			break
		}
		c.peers[from.String()] = nil
	case dht.PEERS:
		if !c.peerKnown(from) {
			errs = append(errs, fmt.Errorf("PEERS recieved from unknown peer %s, adding this peer", from.String()))
			c.addPeerUnchecked(from)
		}
		JSONpeers := []string{}
		err = json.Unmarshal(payload, &JSONpeers)
		if err != nil {
			errs = append(errs, err)
			break
		}
		for _, p := range JSONpeers {
			if _, ok := c.peers[from.String()]; !ok {
				c.peers[p] = &Peer{*from, make(map[dht.Key]bool), time.Now()}
			}
		}
	case dht.KEYS:
		if !c.peerKnown(from) {
			errs = append(errs, fmt.Errorf("KEYS recieved from unknown peer %s, adding this peer", from.String()))
			c.addPeerUnchecked(from)
		}
		JSONKeys := [][64]byte{}
		err = json.Unmarshal(payload, &JSONKeys)
		if err != nil {
			errs = append(errs, err)
			break
		}
		for _, k := range JSONKeys {
			if _, ok := c.peers[from.String()].peerkeys[k]; !ok {
				c.peers[from.String()].peerkeys[k] = true
			}
		}
	case dht.VALUE:
		if !c.peerKnown(from) {
			errs = append(errs, fmt.Errorf("VALUE recieved from unknown peer %s, adding this peer", from.String()))
			c.addPeerUnchecked(from)
		}
		v, err := DeserializeValue(payload)
		if err != nil {
			errs = append(errs, err)
			break
		}
		c.table[v.Hash] = &v
	default:
		errs = append(errs, fmt.Errorf("unknown command %d recieved from %s", cmd, from.String()))
	}

	if c.peerKnown(from) {
		c.peers[from.String()].lastseen = time.Now()
	}

	return errors.Join(errs...)
}

func (c *Client) peerKnown(from *net.UDPAddr) bool {
	_, ok := c.peers[from.String()]
	return ok
}

func (c *Client) addPeerUnchecked(from *net.UDPAddr) {
	c.peers[from.String()] = &Peer{*from, make(map[dht.Key]bool), time.Now()}
}

func DeserializeValue(serialized []byte) (v dht.Value, err error) {
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
