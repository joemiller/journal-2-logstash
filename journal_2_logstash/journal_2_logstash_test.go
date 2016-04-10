package journal_2_logstash

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var journalMessageExample = `{ "__CURSOR" : "s=61821e0261d64e798421262e919e98c2;i=e954abb;b=02af341160dc4db3a323457d05bac86d;m=69a8e78f37b;t=52a6d993c7998;x=152f4cdf1f7a5704", "__REALTIME_TIMESTAMP" : "1454025094232472", "__MONOTONIC_TIMESTAMP" : "7260885021563", "_BOOT_ID" : "02af341160dc4db3a323457d05bac86d", "MESSAGE" :"foo" }`
var expectedCursor = "s=61821e0261d64e798421262e919e98c2;i=e954abb;b=02af341160dc4db3a323457d05bac86d;m=69a8e78f37b;t=52a6d993c7998;x=152f4cdf1f7a5704"

func Test_cursorFromRawMessage(t *testing.T) {
	cursor := cursorFromRawMessage([]byte(journalMessageExample))
	assert.Equal(t, expectedCursor, cursor)
}

func Test_cursorFromRawMessage_corruptMessage(t *testing.T) {
	journalMessageExample := `{  "s=61821e0261d64e798421262e919e" }`
	expectedCursor := ""

	cursor := cursorFromRawMessage([]byte(journalMessageExample))
	assert.Equal(t, expectedCursor, cursor)
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

	// test 1 - a call to saveCursor() with no message received should return immediately
	//          with no side effects (ie: the jtls.lastMessage field should be empty/zero-value)
	err := jtls.saveCursor()
	assert.Nil(t, err)

	// test 2 - saveCursor() should extract the cursor from jtls.lastMessage and save it to our
	//          tempfile.
	jtls.lastMessage = []byte(journalMessageExample)
	err = jtls.saveCursor()
	assert.Nil(t, err)

	savedValue, _ := readStateFile(jtls.StateFile)
	assert.Equal(t, savedValue, expectedCursor)

	// test 3 - saveCursor is unable to extract a cursor. Possibly garbled/corrupt log message?
	jtls.lastMessage = []byte("foobar")
	err = jtls.saveCursor()
	assert.Equal(t, err, errors.New("Unable to get cursor from most recent log message."))

}

func TestMetrics(t *testing.T) {
	m := newMetrics()
	s := &JournalShipper{journalMetrics: m}

	s.msgsRecvd.Inc(42)
	assert.Equal(t, int64(42), s.msgsRecvd.Count())
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
