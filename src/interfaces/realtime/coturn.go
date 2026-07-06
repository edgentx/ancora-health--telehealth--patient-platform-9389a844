package realtime

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- coturn's REST-API long-term-credential scheme (RFC 5389/8489) mandates HMAC-SHA1; the digest authenticates a time-boxed TURN username, not confidentiality.
	"encoding/base64"
	"strconv"
	"time"
)

// TurnCredentials is a short-lived credential a client uses to authenticate to
// the coturn TURN/STUN servers for NAT traversal during a video visit. The
// gateway hands these to a peer at the start of a signaling session; the client
// feeds them into its RTCPeerConnection iceServers configuration.
type TurnCredentials struct {
	// Username is the ephemeral TURN username, "<expiry-unix>:<userID>". coturn
	// parses the leading timestamp to reject expired credentials.
	Username string `json:"username"`
	// Password is base64(HMAC-SHA1(sharedSecret, Username)), the value coturn
	// recomputes and compares under its use-auth-secret (REST API) mode.
	Password string `json:"credential"`
	// TTLSeconds is how long the credential remains valid, echoed for the client.
	TTLSeconds int `json:"ttl"`
	// URIs are the TURN/STUN server URIs the client should try, e.g.
	// "turn:turn.{project}.vforce360.ai:3478".
	URIs []string `json:"urls"`
}

// CoturnIssuer mints ephemeral TURN credentials using coturn's long-term-secret
// (REST API) scheme. The shared secret matches coturn's `static-auth-secret`, so
// no per-user password is provisioned: the credential is a time-boxed HMAC any
// coturn replica configured with the same secret can validate statelessly.
type CoturnIssuer struct {
	secret string
	uris   []string
	ttl    time.Duration
	now    func() time.Time
}

// NewCoturnIssuer builds an issuer bound to coturn's shared auth secret and the
// TURN/STUN URIs to advertise. A non-positive ttl falls back to one hour, the
// conventional coturn REST credential lifetime.
func NewCoturnIssuer(secret string, uris []string, ttl time.Duration) *CoturnIssuer {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &CoturnIssuer{secret: secret, uris: uris, ttl: ttl, now: time.Now}
}

// Issue derives an ephemeral credential valid for the issuer's TTL from the
// current time, binding the username to userID so coturn can attribute usage.
func (c *CoturnIssuer) Issue(userID string) TurnCredentials {
	expiry := c.now().Add(c.ttl).Unix()
	username := strconv.FormatInt(expiry, 10) + ":" + userID

	mac := hmac.New(sha1.New, []byte(c.secret))
	_, _ = mac.Write([]byte(username))
	password := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return TurnCredentials{
		Username:   username,
		Password:   password,
		TTLSeconds: int(c.ttl.Seconds()),
		URIs:       c.uris,
	}
}
