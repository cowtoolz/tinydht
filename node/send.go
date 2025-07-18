package node

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/cowtools/tinydht/dht"
)

func (c *Client) Broadcast() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	errs := []error{}

	needsPruned := make(map[string]bool)
	for pStr, p := range c.peers {
		if _, ok := needsPruned[pStr]; ok {
			continue
		}
		err := c.sendKEYS(&p.addr)
		if err != nil {
			errs = append(errs, fmt.Errorf("error while sending keys: %s", err))
		}
		err = c.sendPEERS(&p.addr)
		if err != nil {
			errs = append(errs, fmt.Errorf("error while sending peers: %s", err))
		}
		for k, v := range c.table {
			if v.Expires.Before(time.Now()) {
				c.table[k] = nil
				continue
			}
			if p.lastseen.Before(time.Now().Add(-time.Minute)) {
				needsPruned[pStr] = true
			}
			if !p.peerkeys[k] {
				err = c.sendVALUE(&p.addr, v)
				if err != nil {
					errs = append(errs, fmt.Errorf("error while sending value: %s", err))
				}
			}
		}
	}

	// Prune peers that have timed out
	for k := range needsPruned {
		c.peers[k] = nil
	}

	return errors.Join(errs...)

}

func (c *Client) sendCommand(p *net.UDPAddr, cmd dht.Command, payload []byte) error {
	var data []byte
	if payload == nil {
		data = []byte{cmd}
	} else {
		data = append([]byte{cmd}, payload...)
	}
	_, err := c.listener.WriteToUDP(data, p)
	return err
}

func (c *Client) sendHELLO(p *net.UDPAddr) error {
	return c.sendCommand(p, dht.HELLO, nil)
}

func (c *Client) sendGOODBYE(p *net.UDPAddr) error {
	return c.sendCommand(p, dht.GOODBYE, nil)
}

func (c *Client) sendPEERS(p *net.UDPAddr) error {
	peers := []string{}
	for p := range c.peers {
		peers = append(peers, p)
	}
	peersJSON, err := json.Marshal(peers)
	if err != nil {
		return err
	}
	return c.sendCommand(p, dht.PEERS, peersJSON)
}

func (c *Client) sendKEYS(p *net.UDPAddr) error {
	keys := []dht.Key{}
	for k := range c.table {
		keys = append(keys, k)
	}
	keysJSON, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	return c.sendCommand(p, dht.KEYS, keysJSON)
}

func (c *Client) sendVALUE(p *net.UDPAddr, payload *dht.Value) error {
	return c.sendCommand(p, dht.VALUE, serialize(*payload))
}

// Takes a value and serializes it for sharing
func serialize(v dht.Value) (serialized []byte) {
	serialized = binary.BigEndian.AppendUint64(serialized, uint64(v.Expires.Unix()))
	serialized = binary.BigEndian.AppendUint64(serialized, uint64(len(v.Payload)))
	serialized = append(serialized, v.Hash[:]...)
	serialized = append(serialized, v.Payload...)
	return serialized
}
