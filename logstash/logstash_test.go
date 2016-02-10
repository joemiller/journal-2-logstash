package logstash

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/pantheon-systems/journal-2-logstash/tlstest"
	"github.com/stretchr/testify/assert"
)

var (
	client *Client
	server *tlstest.Server
)

func makeTlsConfigFromFiles(keyFile, certFile, caFile string) (*tls.Config, error) {
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
	return tlsConfig, nil
}

func setup(t *testing.T) {
	var err error

	// setup mock logstash TLS (basic tcp socket) server
	serverTlsConfig, err := makeTlsConfigFromFiles("../test/fixtures/certs/logstash.key",
		"../test/fixtures/certs/logstash.crt",
		"../test/fixtures/certs/ca.crt")
	assert.Nil(t, err)
	server, err = tlstest.NewServer(serverTlsConfig)
	assert.Nil(t, err)

	// setup logstash tls client
	client, err = NewClient(server.Address(),
		"../test/fixtures/certs/logger.key",
		"../test/fixtures/certs/logger.crt",
		"../test/fixtures/certs/ca.crt")
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestWrite(t *testing.T) {
	setup(t)
	defer server.Close()
	defer client.Close()

	client.Write([]byte("pretend this is JSON"))
	server.WaitForLines(1, time.Second)
	assert.True(t, server.Received("pretend this is JSON"))
}
