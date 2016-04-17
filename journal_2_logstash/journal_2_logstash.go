package journal_2_logstash

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/pantheon-systems/journal-2-logstash/journal"
	"github.com/pantheon-systems/journal-2-logstash/logstash"
	"github.com/rcrowley/go-metrics"
)

var (
	saveInterval = float64(30) // seconds  // TODO: make this configurable
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
	msgsRecvd metrics.Counter
	msgsSent  metrics.Counter
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
		// TODO: we're going to need hostnames on these metrics since this agent will run on a lot of nodes.
		//go graphite.Graphite(metrics.DefaultRegistry, 60*time.Second, "journal-2-logstash", addr)
	}

	return s, nil
}

func newMetrics() journalMetrics {
	m := journalMetrics{
		msgsRecvd: metrics.NewCounter(),
		msgsSent:  metrics.NewCounter(),
	}
	metrics.Register("messages_received", m.msgsRecvd)
	metrics.Register("messages_sent", m.msgsSent)
	return m
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

	log.Printf("Saving cursor to %s: %v", s.StateFile, cursor)
	if err := writeStateFile(s.StateFile, cursor); err != nil {
		return fmt.Errorf("Unable to write to state file: %s", err.Error())
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
	var i interface{}
	json.Unmarshal(*raw, &i)

	m := i.(map[string]interface{})
	for k, v := range m {
		switch k {
		case "__REALTIME_TIMESTAMP":
			val, err := strconv.ParseInt(v.(string), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Unable to parse __REALTIME_TIMESTAMP from Journal message: %s", err)
			}
			t := logstash.JournalTime(val)
			e.Timestamp = t
		case "MESSAGE":
			e.Message = v.(string)
		default:
			e.Fields[k] = v.(string)
		}
	}
	return e, nil
}

func (s *JournalShipper) Run() error {
	logsCh, err := s.journal.Follow()
	if err != nil {
		return fmt.Errorf("Error reading from systemd-journal-gatewayd: %s", err.Error())
	}

	// loop forever receiving messages from s-j-gatewayd and relaying them to logstash
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
			s.msgsRecvd.Inc(1)

			event, err := logstashEventFromJournal(&rawMessage)
			if err != nil {
				// TODO: increment corrupt_msg counter
				log.Printf("Error parsing log: %s", err)
				continue
			}

			if _, err := s.logstash.Write(event); err != nil {
				return fmt.Errorf("Error writing to logstash: %s", err)
			}
			s.msgsSent.Inc(1)

			if time.Since(s.lastStateSave).Seconds() > saveInterval {
				if err := s.saveCursor(event.Fields["__CURSOR"]); err != nil {
					return fmt.Errorf("Error saving cursor: %s", err)
				}
			}
		}
	}
}
