package logstash

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"

	"github.com/cenkalti/backoff"
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

func (c *Client) Write(e *V1Event) (int, error) {
	bytes, err := e.ToJSON()
	if err != nil {
		return 0, err
	}
	return c.writeAndRetry(append(bytes, '\n'))
}

func (c *Client) connect() error {
	c.Close()
	var err error
	conn := &tls.Conn{}

	operation := func() error {
		conn, err = tls.Dial("tcp", c.url, c.config)
		if err != nil {
			log.Printf("Error connecting to logstash: %s", err)
		}
		return err
	}
	err = backoff.Retry(operation, backoff.NewExponentialBackOff())
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

// TODO: we have a retry in the connect() func, should we also retry in the write path?
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
