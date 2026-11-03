package xsock

import (
	"bytes"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func unixPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "xsock-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	addr := &net.UnixAddr{Name: filepath.Join(dir, "test.sock"), Net: "unix"}
	l, err := net.ListenUnix("unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	client, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	server, err := l.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	return client, server
}

func runHandler(t *testing.T, conn *net.UnixConn) (<-chan []byte, <-chan struct{}, *bytes.Buffer) {
	t.Helper()
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	s := CreateXSockServer(&Config{ByteBufferSize: 4})
	results := make(chan []byte, 4)
	done := make(chan struct{})
	go func() { s.connectionHandler(conn, &results); close(done) }()
	return results, done, &output
}

func awaitClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("connection handler did not terminate")
	}
}

func TestPeerCloseDoesNotLogReadError(t *testing.T) {
	client, server := unixPair(t)
	_, done, output := runHandler(t, server)
	client.Close()
	awaitClosed(t, done)
	if output.Len() != 0 {
		t.Fatalf("normal connection close logged as error: %s", output.String())
	}
}

func TestFramedMessagesAreDeliveredBeforeQuietEOF(t *testing.T) {
	client, server := unixPair(t)
	results, done, output := runHandler(t, server)
	want := []string{`{"kind":"first"}`, `{"kind":"second"}`}
	for _, payload := range want {
		if _, err := client.Write(append([]byte(payload), 0x03, 0x03)); err != nil {
			t.Fatal(err)
		}
	}
	client.Close()
	awaitClosed(t, done)
	for _, payload := range want {
		select {
		case got := <-results:
			if string(got) != payload {
				t.Fatalf("payload changed: got %q want %q", got, payload)
			}
		default:
			t.Fatal("framed message was dropped")
		}
	}
	if output.Len() != 0 {
		t.Fatalf("normal connection close logged as error: %s", output.String())
	}
}

func TestRealReadErrorIsStillLogged(t *testing.T) {
	_, server := unixPair(t)
	if err := server.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	_, done, output := runHandler(t, server)
	awaitClosed(t, done)
	if !strings.Contains(output.String(), "Message read error:") || !strings.Contains(output.String(), "timeout") {
		t.Fatalf("read timeout was not logged: %s", output.String())
	}
}
