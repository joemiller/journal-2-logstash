package journal_2_logstash

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"sync"
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
	stopCh        chan bool
	wg            *sync.WaitGroup
}

func NewShipper(cfg JournalShipperConfig) (*JournalShipper, error) {
	s := &JournalShipper{
		JournalShipperConfig: cfg,
		stopCh:               make(chan bool),
		wg:                   &sync.WaitGroup{},
	}

	// load "last-sent" cursor from state file, if available
	cursor, err := readStateFile(s.StateFile)
	if cursor != "" {
		log.Printf("Loaded cursor: %s\n", cursor)
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

	fmt.Printf("Saving cursor: %v\n", cursor)
	if err := writeStateFile(s.StateFile, cursor); err != nil {
		return fmt.Errorf("Unable to write to state file: %s", err.Error())
	}
	s.lastStateSave = time.Now()
	return nil
}

func (s *JournalShipper) Run() {
	s.wg.Add(1)
	defer s.wg.Done()

	logsCh, err := s.journal.Follow()
	if err != nil {
		log.Fatalf("Error reading from systemd-journal-gatewayd: %s", err.Error())
	}

	for {
		select {
		case <-s.stopCh:
			return
		case s.lastMessage = <-logsCh:
			if s.Debug {
				log.Printf("[DEBUG] Received from journal: %s\n", s.lastMessage)
			}
			if _, err := s.logstash.Write(s.lastMessage); err != nil {
				log.Fatal(err)
			}
			if time.Since(s.lastStateSave).Seconds() > SAVE_INTERVAL {
				s.saveCursor()
			}
		}
	}
}

func (s *JournalShipper) Stop() {
	log.Printf("Shutting down journal shipper")
	close(s.stopCh)
	s.wg.Wait()
	s.saveCursor()
}
