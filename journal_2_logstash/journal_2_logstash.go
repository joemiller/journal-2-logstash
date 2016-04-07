package journal_2_logstash

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/pantheon-systems/journal-2-logstash/journal"
	"github.com/pantheon-systems/journal-2-logstash/logstash"
	"github.com/rcrowley/go-metrics"
)

var (
	SAVE_INTERVAL = float64(15) // seconds  // TODO: make this configurable
	cursorRegex   = regexp.MustCompile(`"__CURSOR" : "(\S+)"`)
)

type JournalShipperConfig struct {
	Debug       bool
	StateFile   string
	Socket      string
	Url         string
	Key         string
	Cert        string
	Ca          string
	GraphiteUrl string
}

type JournalShipper struct {
	JournalShipperConfig
	lastStateSave time.Time
	journal       *journal.Journal // TODO: rename this to journal.Follower() ?
	logstash      *logstash.Client
	lastMessage   []byte
	journalMetrics
}

type journalMetrics struct {
	msgsRecvd metrics.Counter
	msgsSent  metrics.Counter
}

func NewShipper(cfg JournalShipperConfig) (*JournalShipper, error) {
	m := newMetrics()
	s := &JournalShipper{
		JournalShipperConfig: cfg,
		journalMetrics:       m,
	}

	// load "last-sent" cursor from state file, if available
	cursor, err := readStateFile(s.StateFile)
	if cursor != "" {
		log.Printf("Loaded cursor: %s from %s", cursor, s.StateFile)
	} else {
		log.Printf("Could not load cursor (%v). Will start reading from 'last boot time'.", err)
	}

	// open the journal
	s.journal, err = journal.NewJournal(cursor, s.Socket)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to systemd-journal-gatewayd: %s", err.Error())
	}

	// connect to logstash TLS
	s.logstash, err = logstash.NewClient(s.Url, s.Key, s.Cert, s.Ca)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to logstash: %s", err.Error())
	}

	// setup periodic metric logging to stderr
	go metrics.Log(metrics.DefaultRegistry, 60*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	// also send metrics to graphite if a GraphiteUrl config option was specified
	if cfg.GraphiteUrl != "" {
		addr, err := net.ResolveTCPAddr("tcp", cfg.GraphiteUrl)
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

// takes a raw json message representing a log record for s-j-gatewayd and extracts the '__CURSOR' record
// using a regex match. We use a regex to avoid having to unmarshal the JSON and re-marshal it before sending
// on to logstash
func cursorFromRawMessage(log []byte) string {
	cursor := cursorRegex.FindStringSubmatch(string(log))
	if len(cursor) > 0 {
		return cursor[1]
	}
	return ""
}

// persist the last read cursor to the state file
func (s *JournalShipper) saveCursor() error {
	if len(s.lastMessage) == 0 {
		return nil
	}

	cursor := cursorFromRawMessage(s.lastMessage)
	if cursor == "" {
		return fmt.Errorf("Unable to get cursor from most recent log message.")
	}

	log.Printf("Saving cursor: %v to %s\n", cursor, s.StateFile)
	if err := writeStateFile(s.StateFile, cursor); err != nil {
		return fmt.Errorf("Unable to write to state file: %s", err.Error())
	}
	s.lastStateSave = time.Now()
	return nil
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
		case s.lastMessage = <-logsCh:
			s.msgsRecvd.Inc(1)
			if s.Debug {
				fmt.Printf("[DEBUG] Received from journal: %s\n", s.lastMessage)
			}
			// channel is closed, we're done
			if len(s.lastMessage) == 0 {
				return fmt.Errorf("systemd-journal-gatewayd connection closed.")
			}
			if _, err := s.logstash.Write(s.lastMessage); err != nil {
				return fmt.Errorf("Error writing to logstash: %s\n", err)
			}
			s.msgsSent.Inc(1)
			if time.Since(s.lastStateSave).Seconds() > SAVE_INTERVAL {
				s.saveCursor()
			}
		}
	}
}
