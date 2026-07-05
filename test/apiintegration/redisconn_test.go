package apiintegration

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// readRESPCommand reads one RESP command (an array of bulk strings) off the
// connection and returns it as space-joined arguments, so a test can assert on
// the exact wire form the client produced.
func readRESPCommand(r *bufio.Reader) (string, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	n, err := strconv.Atoi(strings.TrimRight(header[1:], "\r\n")) // *<n>
	if err != nil {
		return "", err
	}
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		lenLine, err := r.ReadString('\n') // $<len>
		if err != nil {
			return "", err
		}
		l, err := strconv.Atoi(strings.TrimRight(lenLine[1:], "\r\n"))
		if err != nil {
			return "", err
		}
		buf := make([]byte, l+2) // payload + CRLF
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		args = append(args, string(buf[:l]))
	}
	return strings.Join(args, " "), nil
}

// scriptedServer plays a Redis server over one end of a net.Pipe: it reads the
// client's command, hands it back on the returned channel, then writes a canned
// reply. It verifies the codec (command encoding + reply parsing) with no live
// Redis.
func scriptedServer(t *testing.T, server net.Conn, reply string) chan string {
	t.Helper()
	got := make(chan string, 1)
	go func() {
		defer server.Close()
		cmd, err := readRESPCommand(bufio.NewReader(server))
		if err != nil {
			got <- "ERR:" + err.Error()
			return
		}
		if _, err := server.Write([]byte(reply)); err != nil {
			got <- "ERR:" + err.Error()
			return
		}
		got <- cmd
	}()
	return got
}

func TestRedisConn_SetNXAcquired(t *testing.T) {
	client, server := net.Pipe()
	got := scriptedServer(t, server, "+OK\r\n")
	c := newRedisConn(client)
	defer c.Close()

	ok, err := c.SetNX(context.Background(), "ancora:slot", "holder-1", 2*time.Second)
	if err != nil {
		t.Fatalf("SetNX: %v", err)
	}
	if !ok {
		t.Fatal("SetNX returned false on +OK, want true (acquired)")
	}
	if cmd := <-got; cmd != "SET ancora:slot holder-1 NX PX 2000" {
		t.Fatalf("wire command = %q, want the SET…NX PX form", cmd)
	}
}

func TestRedisConn_SetNXHeld(t *testing.T) {
	client, server := net.Pipe()
	_ = scriptedServer(t, server, "$-1\r\n") // suppressed NX -> nil bulk
	c := newRedisConn(client)
	defer c.Close()

	ok, err := c.SetNX(context.Background(), "ancora:slot", "holder-2", time.Second)
	if err != nil {
		t.Fatalf("SetNX: %v", err)
	}
	if ok {
		t.Fatal("SetNX returned true on nil reply, want false (already held)")
	}
}

func TestRedisConn_Get(t *testing.T) {
	client, server := net.Pipe()
	_ = scriptedServer(t, server, "$8\r\nholder-1\r\n")
	c := newRedisConn(client)
	defer c.Close()

	val, err := c.Get(context.Background(), "ancora:slot")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "holder-1" {
		t.Fatalf("Get = %q, want holder-1", val)
	}
}

func TestRedisConn_Del(t *testing.T) {
	client, server := net.Pipe()
	_ = scriptedServer(t, server, ":1\r\n")
	c := newRedisConn(client)
	defer c.Close()

	if err := c.Del(context.Background(), "ancora:slot"); err != nil {
		t.Fatalf("Del: %v", err)
	}
}
