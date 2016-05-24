package journal_2_logstash

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var journalMessageExample = `{ "__CURSOR" : "s=61821e0261d64e798421262e919e98c2;i=e954abb;b=02af341160dc4db3a323457d05bac86d;m=69a8e78f37b;t=52a6d993c7998;x=152f4cdf1f7a5704", "__REALTIME_TIMESTAMP" : "1454025094232472", "__MONOTONIC_TIMESTAMP" : "7260885021563", "_BOOT_ID" : "02af341160dc4db3a323457d05bac86d", "MESSAGE" :"foo" }`

var journalMessageBytesMessageExample = `{ "__CURSOR" : "s=61821e0261d64e798421262e919e98c2;i=e954abb;b=02af341160dc4db3a323457d05bac86d;m=69a8e78f37b;t=52a6d993c7998;x=152f4cdf1f7a5704", "__REALTIME_TIMESTAMP" : "1454025094232472", "__MONOTONIC_TIMESTAMP" : "7260885021563", "_BOOT_ID" : "02af341160dc4db3a323457d05bac86d", "COMMAND" : [ 102, 111, 111 ], "MESSAGE" : [ 102, 111, 111 ] }`

var expectedCursor = "s=61821e0261d64e798421262e919e98c2;i=e954abb;b=02af341160dc4db3a323457d05bac86d;m=69a8e78f37b;t=52a6d993c7998;x=152f4cdf1f7a5704"

func Test_logstashEventFromJournal__message_as_string(t *testing.T) {
	raw := []byte(journalMessageExample)
	e, err := logstashEventFromJournal(&raw)
	assert.Nil(t, err)
	assert.Equal(t, "2016-01-28 23:51:34.000232472 +0000 UTC", e.Timestamp.String())
	assert.Equal(t, "foo", e.Message)
	assert.Equal(t, "7260885021563", e.Fields["__MONOTONIC_TIMESTAMP"])
}

func Test_logstashEventFromJournal__message_as_byte_array(t *testing.T) {
	raw := []byte(journalMessageBytesMessageExample)
	e, err := logstashEventFromJournal(&raw)
	t.Log(e)
	assert.Nil(t, err)
	assert.Equal(t, "2016-01-28 23:51:34.000232472 +0000 UTC", e.Timestamp.String())
	assert.Equal(t, "foo", e.Message)
	assert.Equal(t, "foo", e.Fields["COMMAND"])
	assert.Equal(t, "7260885021563", e.Fields["__MONOTONIC_TIMESTAMP"])
}

func Test_timeFromJournalInt(t *testing.T) {
	r := timeFromJournalInt(1460858962473842)
	text, _ := r.MarshalText()
	assert.Equal(t, []byte("2016-04-17T02:09:22.000473842Z"), text)
}

func tempStateFile(t *testing.T) *os.File {
	tempFile, err := ioutil.TempFile("", "journal_2_logstash_tests")
	assert.Nil(t, err)
	return tempFile
}

func Test_saveCursor(t *testing.T) {
	stateFile := tempStateFile(t)
	defer os.Remove(stateFile.Name())

	jtls := &JournalShipper{}
	jtls.StateFile = stateFile.Name()

	// test 1 - a call to saveCursor() empty cursor should return without updating
	//          the lastStateSave time
	tsBefore := jtls.lastStateSave
	err := jtls.saveCursor("")
	assert.Nil(t, err)
	assert.Equal(t, tsBefore, jtls.lastStateSave)

	// test 2 - saveCursor() should save the given cursor and update the lastStateSave
	//          timestamp in the jtls struct.
	tsBefore = jtls.lastStateSave
	err = jtls.saveCursor("foo")
	assert.Nil(t, err)
	assert.NotEqual(t, tsBefore, jtls.lastStateSave)

	savedValue, _ := readStateFile(jtls.StateFile)
	assert.Equal(t, "foo", savedValue)
}

func TestMetrics(t *testing.T) {
	m := newMetrics()
	s := &JournalShipper{journalMetrics: m}

	s.msgsRead.Inc(42)
	assert.Equal(t, int64(42), s.msgsRead.Count())
}

//func Test_Run(t *testing.T) {
//	// setup a fake journal, and fake TLS receiver
//	// test save is called when lastsave>SAVEINTERVAL
//	sockFile := fmt.Sprintf("/tmp/%.sock", uuid.NewV4())
//	unixSocketListener, err := net.Listen("unix", sockFile)
//	assert.Nil(t, err)
//	defer os.Remove(sockFile.Name())
//
//	mux := http.NewServeMux()
//	server = &httptest.Server{
//		Listener: unixSocketListener,
//		Config:   &http.Server{Handler: mux},
//	}
//
//}
