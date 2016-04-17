package logstash

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	referenceTimeInt    = 1460858962473842
	referenceTimeString = "2016-04-17T02:09:22.000473842Z"
)

func referenceEvent() *V1Event {
	event := NewV1Event()
	event.Timestamp = JournalTime(referenceTimeInt)
	event.Message = "foo"
	event.Fields["baz"] = "blah"
	return event
}

func Test_NewV1Event(t *testing.T) {
	event := NewV1Event()
	assert.Equal(t, int32(1), event.Version)
	assert.Equal(t, "", event.Message)
	assert.NotZero(t, event.Timestamp)
	// make sure we're using UTC stamps
	zone, _ := event.Timestamp.Zone()
	assert.Equal(t, "UTC", zone)
}

func Test_Journaltime(t *testing.T) {
	r := JournalTime(referenceTimeInt)
	text, _ := r.MarshalText()
	assert.Equal(t, []byte(referenceTimeString), text)
}

func Test_ToJSON(t *testing.T) {
	event := referenceEvent()

	expected := fmt.Sprintf("{\"@timestamp\":\"%s\",\"@version\":1,\"baz\":\"blah\",\"message\":\"foo\"}", referenceTimeString)
	actual, err := event.ToJSON()
	t.Logf("%s", actual)
	assert.Nil(t, err)
	assert.Equal(t, []byte(expected), actual)
}
