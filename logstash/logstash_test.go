package logstash

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
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

func makeTLSConfigFromFiles(keyFile, certFile, caFile string) (*tls.Config, error) {
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

func setup(t *testing.T, timeout time.Duration) {
	var err error

	// setup mock logstash TLS (basic tcp socket) server
	serverTLSConfig, err := makeTLSConfigFromFiles("../test/fixtures/certs/logstash.key",
		"../test/fixtures/certs/logstash.crt",
		"../test/fixtures/certs/ca.crt")
	assert.Nil(t, err)
	server, err = tlstest.NewServer(serverTLSConfig)
	assert.Nil(t, err)

	// setup logstash tls client
	client, err = NewClient(server.Address(),
		"../test/fixtures/certs/logger.key",
		"../test/fixtures/certs/logger.crt",
		"../test/fixtures/certs/ca.crt",
		timeout)
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func teardown() {
	client.Close()
	server.Close()
}

func TestWrite(t *testing.T) {
	setup(t, time.Duration(5*time.Second))
	defer teardown()
	event := referenceEvent()

	client.Write(event)
	server.WaitForLines(1, time.Second)
	t.Log(server.Lines())

	expected := fmt.Sprintf("{\"@timestamp\":\"%s\",\"@version\":1,\"extra_field\":\"text here\",\"message\":\"foo\"}", referenceTimeString)
	assert.True(t, server.Received(expected))
}

func TestWriteTimeout(t *testing.T) {
	setup(t, time.Duration(-5*time.Second))
	defer teardown()
	event := referenceEvent()

	_, err := client.Write(event)
	t.Log(err)
	assert.NotNil(t, err)
}

func TestPeriodicDisconnect(t *testing.T) {
	setup(t, time.Duration(5*time.Second))
	defer teardown()
	event := referenceEvent()

	// set connection timestamp to 5 minutes in the past
	initialTime := time.Now().Add(-300 * time.Second)
	client.lastConnectTime = initialTime

	// upon next Write(), the connection should be disconnected / reconnected, causing the lastConnectTime to be updated
	client.Write(event)
	assert.NotEqual(t, initialTime, client.lastConnectTime)
}
