package util

// Call fn repeatedly until an error is returned; then send the error
// on the given channel and return
func errToChannel(fn func() error, ch chan <- error) {
	var err error
	for err = fn(); err == nil; err = fn() {}
	ch <- err
}



type SSLMode string

const (
	SSLDisable SSLMode = "disable"
	SSLAllow           = "allow"
	SSLPrefer          = "prefer"
	SSLRequire         = "require"
)

type SSLConfig struct {
	tls.Config
	Mode SSLMode
}

func NegotiateTLS(c net.Conn, config *SSLConfig) (net.Conn, error) {
	sslMode := config.Mode
	if sslmode != SSLDisable {
		// send an SSLRequest message
		// length: int32(8)
		// code:   int32(80877103)
		c.Write([]byte{0x00, 0x00, 0x00, 0x08,
			0x04, 0xd2, 0x16, 0x2f})

		sslResponse := make([]byte, 1)
		bytesRead, err := io.ReadFull(c, sslResponse)
		if bytesRead != 1 || err != nil {
			return nil, errors.New("Could not read response to SSL Request")
		}

		if sslResponse[0] == 'S' {
			return tls.Client(c, config), nil
		} else if sslResponse[0] == 'N' && sslmode != SSLAllow &&
			sslmode != SSLPrefer {
			// reject; we require ssl
			return nil, errors.New("SSL required but declined by server.")
		} else {
			return c, nil
		}

		panic("Oh snap!")
	}

	return c, nil
}
