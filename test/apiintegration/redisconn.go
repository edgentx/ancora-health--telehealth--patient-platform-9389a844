package apiintegration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// redisConn is a dependency-free Redis client speaking just enough of the RESP
// protocol to back locking.RedisConn: SET … NX PX, GET, and DEL. It lets the
// suite exercise the production RedisSlotLocker against a live Redis without
// pulling a full Redis driver into the module's dependency graph. Access is
// serialized so a single connection is safe under the concurrent bookings the
// double-book test issues.
type redisConn struct {
	mu   sync.Mutex
	conn net.Conn
	rw   *bufio.ReadWriter
}

// dialRedis opens a connection to a Redis server at addr (host:port).
func dialRedis(addr string) (*redisConn, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return newRedisConn(conn), nil
}

// newRedisConn wraps an established connection. It is separated from dialRedis
// so the protocol codec can be exercised over an in-memory net.Pipe without a
// live server.
func newRedisConn(conn net.Conn) *redisConn {
	return &redisConn{
		conn: conn,
		rw:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

// Close releases the underlying connection.
func (c *redisConn) Close() error { return c.conn.Close() }

// SetNX issues SET key value NX PX <ttl-ms>. Redis replies +OK when the key was
// set and a nil bulk when it already existed, which map to the (true/false)
// acquire semantics the slot locker expects.
func (c *redisConn) SetNX(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	ms := ttl.Milliseconds()
	if ms <= 0 {
		ms = int64(defaultHoldTTLMillis)
	}
	reply, err := c.do("SET", key, value, "NX", "PX", strconv.FormatInt(ms, 10))
	if err != nil {
		return false, err
	}
	// A successful SET returns the simple string "OK"; a suppressed NX returns a
	// nil bulk (reply == nil).
	if s, ok := reply.(string); ok && s == "OK" {
		return true, nil
	}
	return false, nil
}

// Get issues GET key, returning ("", nil) when the key is absent.
func (c *redisConn) Get(_ context.Context, key string) (string, error) {
	reply, err := c.do("GET", key)
	if err != nil {
		return "", err
	}
	if reply == nil {
		return "", nil
	}
	if s, ok := reply.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("redis: unexpected GET reply %T", reply)
}

// Del issues DEL key, discarding the count of removed keys.
func (c *redisConn) Del(_ context.Context, key string) error {
	_, err := c.do("DEL", key)
	return err
}

// do writes a RESP command as an array of bulk strings and reads one reply.
func (c *redisConn) do(args ...string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := writeCommand(c.rw.Writer, args); err != nil {
		return nil, err
	}
	if err := c.rw.Writer.Flush(); err != nil {
		return nil, err
	}
	return readReply(c.rw.Reader)
}

// writeCommand encodes args as a RESP array of bulk strings.
func writeCommand(w *bufio.Writer, args []string) error {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, a := range args {
		if _, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(a), a); err != nil {
			return err
		}
	}
	return nil
}

// readReply parses a single RESP reply. It returns a string for simple and bulk
// strings, an int64 for integers, nil for a nil bulk, and an error for a RESP
// error reply. It supports only the reply shapes the three commands above can
// produce.
func readReply(r *bufio.Reader) (any, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, fmt.Errorf("redis: empty reply")
	}
	prefix, rest := line[0], line[1:]
	switch prefix {
	case '+': // simple string
		return rest, nil
	case '-': // error
		return nil, fmt.Errorf("redis: %s", rest)
	case ':': // integer
		return strconv.ParseInt(rest, 10, 64)
	case '$': // bulk string
		n, err := strconv.Atoi(rest)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil // nil bulk
		}
		buf := make([]byte, n+2) // payload + CRLF
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	default:
		return nil, fmt.Errorf("redis: unsupported reply prefix %q", string(prefix))
	}
}

// readLine reads a single CRLF-terminated protocol line, without the trailing
// CRLF.
func readLine(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}

// defaultHoldTTLMillis mirrors locking.DefaultHoldTTL in milliseconds,
// used when a caller passes a non-positive ttl.
const defaultHoldTTLMillis = 5 * 60 * 1000
