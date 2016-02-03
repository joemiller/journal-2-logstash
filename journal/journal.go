package journal

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
)

// TODO: rename this package to journal_follower ?

type Journal struct {
	Cursor string
	Socket string
	Client *http.Client
}

func makeUnixSocketTransport(sock string) *http.Transport {
	return &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", sock)
		},
	}
}

func NewJournal(cursor, socket string) (*Journal, error) {
	// TODO: we could support IP transport in addition to unix sockets. The default
	//       configuration of s-j-gatewayd uses IP sockets but we prefer unix sockets
	//       in order to provide better security to the journal.
	c := &http.Client{
		Transport: makeUnixSocketTransport(socket),
	}

	j := &Journal{
		Cursor: cursor,
		Socket: socket,
		Client: c,
	}
	return j, nil
}

func (j *Journal) makeFollowRequest() (*http.Request, error) {
	req, err := http.NewRequest("GET", "http://journal/entries?follow", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")

	if j.Cursor != "" {
		req.Header.Add("Range", fmt.Sprintf("entries=%s", j.Cursor))
	} else {
		// tail
		req.Header.Add("Range", "entries=:-1:-1")
	}
	return req, nil
}

func (j *Journal) Follow() (<-chan []byte, error) {
	req, err := j.makeFollowRequest()
	if err != nil {
		return nil, err
	}
	resp, err := j.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non 200 response: %d", resp.StatusCode)
	}
	logs := make(chan []byte)
	scanner := bufio.NewScanner(resp.Body)
	go func() {
		for scanner.Scan() {
			data := scanner.Bytes()
			line := make([]byte, len(data))
			copy(line, data)
			logs <- line
		}
		if err := scanner.Err(); err != nil {
			log.Println(err.Error())
		}
	}()
	return logs, nil
}
