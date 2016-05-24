package journal_2_logstash

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/pantheon-systems/journal-2-logstash/journal"
	"github.com/pantheon-systems/journal-2-logstash/logstash"
	"github.com/rcrowley/go-metrics"
)

var (
	saveInterval = time.Duration(30) * time.Second // seconds  // TODO: make this configurable
)

type JournalShipperConfig struct {
	Debug       bool
	StateFile   string
	Socket      string
	URL         string
	Key         string
	Cert        string
	Ca          string
	GraphiteURL string
}

type JournalShipper struct {
	JournalShipperConfig
	lastStateSave time.Time
	journal       *journal.Journal // TODO: rename this to journal.Follower() ?
	logstash      *logstash.Client
	journalMetrics
}

type journalMetrics struct {
	msgsRead      metrics.Counter
	msgsSent      metrics.Counter
	parseFail     metrics.Counter
	secondsBehind metrics.GaugeFloat64
}

func NewShipper(cfg JournalShipperConfig) (*JournalShipper, error) {
	m := newMetrics()
	s := &JournalShipper{
		lastStateSave:        time.Now(),
		JournalShipperConfig: cfg,
		journalMetrics:       m,
	}

	// load "last-sent" cursor from state file, if available
	cursor, err := readStateFile(s.StateFile)
	if cursor != "" {
		log.Printf("Loaded cursor from %s: %s", s.StateFile, cursor)
	} else {
		log.Printf("Could not load cursor (%v). Will start reading from the tail.", err)
	}

	// open the journal
	s.journal, err = journal.NewJournal(cursor, s.Socket)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to systemd-journal-gatewayd: %s", err.Error())
	}

	// connect to logstash TLS
	s.logstash, err = logstash.NewClient(s.URL, s.Key, s.Cert, s.Ca)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to logstash: %s", err.Error())
	}

	// setup periodic metric logging to stderr
	go metrics.Log(metrics.DefaultRegistry, 60*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	// also send metrics to graphite if a GraphiteURL config option was specified
	if cfg.GraphiteURL != "" {
		addr, err := net.ResolveTCPAddr("tcp", cfg.GraphiteURL)
		if err != nil {
			return nil, fmt.Errorf("Invalid graphite-url: %s: %s", addr, err)
		}
		hostname, err := shortHostname()
		if err != nil {
			log.Printf("Unable to determine local hostname. Disabling graphite metrics. %s", err)
		} else {
			metricPath := fmt.Sprintf("servers.%s.journal-2-logstash", hostname) // TODO: expose path to configuration flags
			go graphite.Graphite(metrics.DefaultRegistry, 60*time.Second, metricPath, addr)
		}
	}

	return s, nil
}

func newMetrics() journalMetrics {
	m := journalMetrics{
		msgsRead:      metrics.NewCounter(),
		msgsSent:      metrics.NewCounter(),
		parseFail:     metrics.NewCounter(),
		secondsBehind: metrics.NewGaugeFloat64(),
	}
	metrics.Register("messages_read", m.msgsRead)
	metrics.Register("messages_sent", m.msgsSent)
	metrics.Register("message_parse_fail", m.parseFail)
	metrics.Register("seconds_behind", m.secondsBehind)
	return m
}

func shortHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	index := strings.Index(hostname, ".")
	if index > 0 {
		return hostname[:index], nil
	}
	return hostname, nil
}

func readStateFile(stateFile string) (string, error) {
	cursor, err := ioutil.ReadFile(stateFile)
	return string(cursor), err
}

func writeStateFile(stateFile string, cursor string) error {
	return ioutil.WriteFile(stateFile, []byte(cursor), 0644)
}

// persist the last read cursor to the state file
//
func (s *JournalShipper) saveCursor(cursor string) error {
	if cursor == "" {
		return nil
	}
	if s.Debug {
		log.Printf("Saving cursor to %s: %v", s.StateFile, cursor)
	}
	if err := writeStateFile(s.StateFile, cursor); err != nil {
		return fmt.Errorf("Unable to write state file: %s", err.Error())
	}
	s.lastStateSave = time.Now()
	return nil
}

// logstashEventFromJournal takes a *[]byte containing a raw JSON message from the journal, parses and returns
// a *logstash.V1Event.
//
// The `__REALTIME_TIMESTAMP` field from the journal is converted into Logstash V1Event.Timestmap.
// The `MESSAGE` field from the journal is stored as the Logstash V1Event.Message.
// All other fields are added to Logstash V1Event.Fields[].
//
// The journal encodes all fields as strings and this function assumes all values from the
// parsed JSON will be strings.
//
func logstashEventFromJournal(raw *[]byte) (*logstash.V1Event, error) {
	e := logstash.NewV1Event()
	var msg interface{}
	json.Unmarshal(*raw, &msg)

	m := msg.(map[string]interface{})
	for k, v := range m {
		switch k {
		case "__REALTIME_TIMESTAMP":
			val, err := strconv.ParseInt(v.(string), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Unable to parse __REALTIME_TIMESTAMP from Journal message: %s", err)
			}
			e.Timestamp = timeFromJournalInt(val)
		default:
			err, s := parseJournalValue(v)
			if err != nil {
				return nil, fmt.Errorf("Unable to parse %s field: %s", k, err)
			}
			if k == "MESSAGE" {
				e.Message = s
			} else {
				e.Fields[k] = s
			}
		}
	}
	return e, nil
}

// timeFromJournalInt takes a timestamp (such as __REALTIME_TIMESTAMP) which is
// formatted as microseconds since epoch and returns a golang time.Time
//
// NOTE: this is reduced to millisecond resolution for compatibility with Logstash
//
func timeFromJournalInt(t int64) time.Time {
	secs := t / 1000000
	ms := t % 1000000
	return time.Unix(secs, ms).UTC()
}

// parseJournalValue expects an interface{} containing a field value from the journal which
// can be either a string or array of bytes that will be converted into a string.
//
func parseJournalValue(msg interface{}) (error, string) {
	switch msg := msg.(type) {
	case string:
		return nil, msg
	case []interface{}:
		bytes := make([]byte, len(msg))
		for i := range msg {
			bytes[i] = byte(msg[i].(float64))
		}
		return nil, string(bytes)
	default:
		return errors.New("Journal 'MESSAGE' field of unknown type"), ""
	}
}

func (s *JournalShipper) Run() error {
	logsCh, err := s.journal.Follow()
	if err != nil {
		return fmt.Errorf("Error reading from systemd-journal-gatewayd: %s", err.Error())
	}

	// loop forever reading messages from s-j-gatewayd and relaying them to logstash
	// return with error if we lose connection to the gateway or run into errors sending to logstash
	for {
		select {
		case rawMessage := <-logsCh:
			// channel is closed, we're done
			if len(rawMessage) == 0 {
				return fmt.Errorf("lost connection to systemd-journal-gatewayd")
			}

			if s.Debug {
				log.Printf("[DEBUG] Received from journal: %s", rawMessage)
			}
			s.msgsRead.Inc(1)

			event, err := logstashEventFromJournal(&rawMessage)
			if err != nil {
				log.Printf("Error parsing log: %s", err)
				s.parseFail.Inc(1)
				continue
			}

			if _, err := s.logstash.Write(event); err != nil {
				return fmt.Errorf("Error writing to logstash: %s", err)
			}
			s.msgsSent.Inc(1)
			s.secondsBehind.Update(time.Since(event.Timestamp).Seconds())

			if time.Since(s.lastStateSave) > saveInterval {
				if err := s.saveCursor(event.Fields["__CURSOR"]); err != nil {
					return fmt.Errorf("Error saving cursor: %s", err)
				}
			}
		}
	}
}
