package node

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/cowtools/tinydht/dht"
)

func (c *Client) Broadcast() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	errs := []error{}
	for pStr, p := range c.peers {
		err := c.sendKEYS(&p.addr)
		if err != nil {
			errs = append(errs, err)
		}
		err = c.sendPEERS(&p.addr)
		if err != nil {
			errs = append(errs, err)
		}
		for k, v := range c.table {
			if v.Expires.Before(time.Now()) {
				c.table[k] = nil
				continue
			}
			if p.lastseen.Before(time.Now().Add(-time.Minute)) {
				c.peers[pStr] = nil
			}
			if !p.peerkeys[k] {
				err = c.sendVALUE(&p.addr, v)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errors.Join(errs...)

}

func sendCommand(p *net.UDPAddr, cmd dht.Command, payload []byte) error {
	conn, err := net.DialUDP("udp", nil, p)
	if err != nil {
		return err
	}
	if payload == nil {
		_, err := conn.Write([]byte{cmd})
		if err != nil {
			return err
		}
	} else {
		joinedPayload := append([]byte{cmd}, payload...)
		_, err := conn.Write(joinedPayload)
		if err != nil {
			return err
		}
	}
	return conn.Close()
}

func (c *Client) sendHELLO(p *net.UDPAddr) error {
	return sendCommand(p, dht.HELLO, nil)
}

func (c *Client) sendGOODBYE(p *net.UDPAddr) error {
	return sendCommand(p, dht.GOODBYE, nil)
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
	return sendCommand(p, dht.PEERS, peersJSON)
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
	return sendCommand(p, dht.KEYS, keysJSON)
}

func (c *Client) sendVALUE(p *net.UDPAddr, payload *dht.Value) error {
	return sendCommand(p, dht.VALUE, payload.Serialize())
}
