package node

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/cowtools/tinydht/dht"
	"golang.org/x/crypto/blake2b"
)

type debugJSONClient struct {
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
	j := debugJSONClient{
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
