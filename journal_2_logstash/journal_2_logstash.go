package journal_2_logstash

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"time"

	"github.com/pantheon-systems/journal-2-logstash/journal"
	"github.com/pantheon-systems/journal-2-logstash/logstash"
)

var (
	SAVE_INTERVAL = float64(15) // seconds  // TODO: make this configurable
	cursorRegex   = regexp.MustCompile(`"__CURSOR" : "(\S+)"`)
)

type JournalShipperConfig struct {
	Debug     bool
	StateFile string
	Socket    string
	Url       string
	Key       string
	Cert      string
	Ca        string
}

type JournalShipper struct {
	JournalShipperConfig
	lastStateSave time.Time
	journal       *journal.Journal // TODO: rename this to journal.Follower() ?
	logstash      *logstash.Client
	lastMessage   []byte
	msgsRecvd     int
	msgsSent      int
}

func NewShipper(cfg JournalShipperConfig) (*JournalShipper, error) {
	s := &JournalShipper{
		JournalShipperConfig: cfg,
	}

	// load "last-sent" cursor from state file, if available
	cursor, err := readStateFile(s.StateFile)
	if cursor != "" {
		log.Printf("Loaded cursor: %s from %s\n", cursor, s.StateFile)
	} else {
		log.Printf("Could not load cursor (%v). Will start reading from 'last boot time'.\n", err)
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

	return s, nil
}

func readStateFile(stateFile string) (string, error) {
	cursor, err := ioutil.ReadFile(stateFile)
	return string(cursor), err
}

func writeStateFile(stateFile string, cursor string) error {
	return ioutil.WriteFile(stateFile, []byte(cursor), 0644)
}

func cursorFromRawMessage(log []byte) string {
	cursor := cursorRegex.FindStringSubmatch(string(log))
	if len(cursor) > 0 {
		return cursor[1]
	}
	return ""
}

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

func (s *JournalShipper) printStats() {
	log.Printf("Messages received/sent: %d/%d\n", s.msgsRecvd, s.msgsSent)
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
			s.msgsRecvd++
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
			s.msgsSent++
			if time.Since(s.lastStateSave).Seconds() > SAVE_INTERVAL {
				s.saveCursor()
				s.printStats()
			}
		}
	}
}
