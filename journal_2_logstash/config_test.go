package journal_2_logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgs(t *testing.T) {
	args := []string{
		"-s", "/run/journal/sock",
		"-u", "127.0.0.1:1234",
		"-k", "/tls/client.key",
		"-c", "/tls/client.crt",
		"-a", "/tls/ca.crt",
		"-t", "/etc/journal2sock.state",
	}
	err := ConfigFromArgs(args)
	assert.Nil(t, err)
	assert.Equal(t, Config.Debug, false)
	assert.Equal(t, Config.Socket, "/run/journal/sock")
	assert.Equal(t, Config.Url, "127.0.0.1:1234")
}

func TestArgs__MissingRequired(t *testing.T) {
	args := []string{}
	err := ConfigFromArgs(args)
	assert.NotNil(t, err) // error string should list missing required args
}
