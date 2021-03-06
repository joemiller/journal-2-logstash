package logstash

import (
	"encoding/json"
	"time"
)

// V1Event is a representation of the Logstash V1 Event format.
// Example here (I don't think this is the official document though):
//		https://gist.github.com/jordansissel/2996677
//
type V1Event struct {
	Version   int32
	Message   string
	Timestamp time.Time
	Fields    map[string]string
}

// NewV1Event returns a pointer to a Logstash V1Event with Timestamp init'd to time.Now()
//
func NewV1Event() *V1Event {
	e := &V1Event{
		Version:   1,
		Timestamp: time.Now().UTC(),
		Fields:    make(map[string]string),
	}
	return e
}

// SetTimestamp modifies the event's timestamp and ensures zone is set to UTC
//
func (e *V1Event) SetTimestamp(t time.Time) {
	e.Timestamp = t.UTC()
}

// ToJSON returns a JSON-encoded representation of the event
//
func (e *V1Event) ToJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["@version"] = e.Version
	m["@timestamp"] = e.Timestamp
	m["message"] = e.Message

	for k, v := range e.Fields {
		m[k] = v
	}

	return json.Marshal(m)
}
