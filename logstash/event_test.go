package logstash

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	referenceTime       = time.Unix(1515151515, 0)
	referenceTimeString = "2018-01-05T11:25:15Z"
)

func referenceEvent() *V1Event {
	event := NewV1Event()
	event.Message = "foo"
	event.Fields["extra_field"] = "text here"
	event.SetTimestamp(referenceTime)
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

func Test_ToJSON(t *testing.T) {
	event := referenceEvent()

	expected := fmt.Sprintf("{\"@timestamp\":\"%s\",\"@version\":1,\"extra_field\":\"text here\",\"message\":\"foo\"}", referenceTimeString)
	actual, err := event.ToJSON()
	t.Logf(event.Timestamp.Zone())
	t.Logf("%s", actual)
	assert.Nil(t, err)
	assert.Equal(t, []byte(expected), actual)
}
