package node

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/cowtools/tinydht/dht"
	"golang.org/x/crypto/blake2b"
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
}

// Bootstraps the client and begins peering with the root node
func Start() (c *Client, err error) {
	// Stand up listener
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return c, err
	}
	c = &Client{
		peers:    make(map[string]*Peer),
		table:    make(map[dht.Key]*dht.Value),
		listener: conn,
	}

	// Begin peering with the hardcoded root node
	root := net.UDPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 9092,
	}

	err = c.sendHELLO(&root)
	if err != nil {
		return c, err
	}
	c.addPeerUnchecked(&root)

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

	return errors.Join(errs...)
}

type jsonClient struct {
	Local string
	Peers []string
	DHT   map[string]struct {
		Value   string
		Expires string
	}
}

// Hacked together JSON exporting for the client
func (c *Client) DebugJSON() ([]byte, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	peers := []string{}
	for k := range c.peers {
		peers = append(peers, k)
	}
	dht := make(map[string]struct {
		Value   string
		Expires string
	})
	for k, v := range c.table {
		buf := &bytes.Buffer{}
		enc := base64.NewEncoder(base64.RawStdEncoding, buf)
		_, err := enc.Write(k[:])
		if err != nil {
			return []byte{}, err
		}
		err = enc.Close()
		if err != nil {
			return []byte{}, err
		}
		dht[buf.String()] = struct {
			Value   string
			Expires string
		}{string(v.Payload), v.Expires.Format(time.Stamp)}
	}
	j := jsonClient{
		Local: c.listener.LocalAddr().String(),
		Peers: peers,
		DHT:   dht,
	}
	b, err := json.Marshal(j)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func (c *Client) DebugAddToDHT(data string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	digest := blake2b.Sum512([]byte(data))
	c.table[digest] = &dht.Value{
		Hash:    digest,
		Expires: time.Now().Add(24 * time.Hour),
		Payload: []byte(data),
	}
}
