package node

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/cowtools/tinydht/dht"
)

type Peer struct {
	addr     net.UDPAddr
	peerkeys map[dht.Key]bool // hashset
	lastseen time.Time
}

type Client struct {
	peers    map[string]*Peer
	table    map[dht.Key]*dht.Value
	listener *net.UDPConn
	mtx      sync.Mutex

	Closed chan bool
}

var ROOT_ADDR = net.UDPAddr{
	IP:   net.ParseIP("75.139.146.177"),
	Port: 6232,
}

// Bootstraps the client and begins peering with the root node
func Start() (c *Client, err error) {
	// Stand up listener
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 6232,
	})
	// This is sort of hacky, but allows any node to stand itself up as the root
	if err != nil {
		conn, err = net.ListenUDP("udp", nil)
		if err != nil {
			return c, err
		}
	}
	c = &Client{
		peers:    make(map[string]*Peer),
		table:    make(map[dht.Key]*dht.Value),
		listener: conn,
	}

	// Begin peering with the hardcoded root node

	err = c.sendHELLO(&ROOT_ADDR)
	if err != nil {
		return c, err
	}
	c.addPeerUnchecked(&ROOT_ADDR)

	return c, nil
}

// Cleans up resources allocated to the client
func (c *Client) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// we can't return early because we have resources to clean up
	// so, we'll add any errors to a slice and join them at the end
	errs := []error{}
	for _, p := range c.peers {
		err := c.sendGOODBYE(&p.addr)
		if err != nil {
			errs = append(errs, err)
		}
	}
	err := c.listener.Close()
	if err != nil {
		errs = append(errs, err)
	}

	c.peers = nil
	c.table = nil
	c.listener = nil

	c.Closed <- true

	return errors.Join(errs...)
}
