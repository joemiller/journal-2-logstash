package journal

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

var (
	journal *Journal
	server  *httptest.Server
)

func setup(t *testing.T, code int, body string) {
	var err error

	uuid := uuid.NewV4()
	socket := fmt.Sprintf("/tmp/%s.sock", uuid)
	unixSocketListener, err := net.Listen("unix", socket)
	assert.Nil(t, err)

	mux := http.NewServeMux()
	server = &httptest.Server{
		Listener: unixSocketListener,
		Config:   &http.Server{Handler: mux},
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, body)
	})
	server.Start()

	// TODO: add handler to `server`

	journal, err = NewJournal("", socket)
	assert.Nil(t, err)
}

//func TestNewJournal(t *testing.T) {
//	setup(t)
//}

// Test the `makeFollowRequest` function to ensure it is generating the proper request to s-j-gatewayd, such as
// range headers which are necessary for implementing cursor and tail seeking properly.
func TestMakeFollowRequest(t *testing.T) {
	setup(t, 200, "")
	defer server.Close()
	for testVal, expectedRangeHeader := range map[string]string{
		"":    "entries=:-1:-1", // seek to tail of journal
		"foo": "entries=foo",    // seek to a specific cursor
	} {
		journal.Cursor = testVal
		req, err := journal.makeFollowRequest()
		assert.Nil(t, err)

		rangeHeader := req.Header.Get("Range")
		assert.Equal(t, rangeHeader, expectedRangeHeader)

		acceptHeader := req.Header.Get("Accept")
		assert.Equal(t, acceptHeader, "application/json")
	}
}

func TestFollow__MultiLineResponse(t *testing.T) {
	setup(t, 200, "line 1\nline 2\n")
	defer server.Close()

	logs, err := journal.Follow()
	assert.Nil(t, err)

	data := <-logs
	assert.Equal(t, data, []byte("line 1"))

	data = <-logs
	assert.Equal(t, data, []byte("line 2"))
}

func TestFollow__HttpError(t *testing.T) {
	setup(t, 400, "error")
	defer server.Close()

	_, err := journal.Follow()
	assert.Equal(t, err.Error(), "non 200 response: 400")
}
