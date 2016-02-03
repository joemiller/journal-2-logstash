package logstash

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
)

type Client struct {
	conn   *tls.Conn
	config *tls.Config
	url    string
}

func NewClient(url, keyFile, certFile, caFile string) (*Client, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, errors.New("failed to parse CA certs")
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	c := &Client{
		url:    url,
		config: tlsConfig,
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) Write(b []byte) (int, error) {
	return c.writeAndRetry(append(b, '\n'))
}

func (c *Client) connect() error {
	c.Close()
	conn, err := tls.Dial("tcp", c.url, c.config)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) write(b []byte) (int, error) {
	return c.conn.Write(b)
}

// @TODO(joe): should we implement a more advanced retry system or
//             is it sufficient to exit on failure and let the supervisor
//             restart us at the last position?
func (c *Client) writeAndRetry(b []byte) (int, error) {
	if c.conn != nil {
		if n, err := c.write(b); err == nil {
			return n, err
		}
	}
	if err := c.connect(); err != nil {
		return 0, err
	}
	return c.write(b)
}
