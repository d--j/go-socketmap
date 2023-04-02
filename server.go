package socketmap

import (
	"context"
	"net"
	"strings"
	"time"
)

// A Handler responds to a Sendmail socketmap request.
// ctx is a context you can pass down to potentially long-running functions.
// lookup and key are the two parsed request variables (the map name and the map key to look up).
//
// A Handler should return found = true when key was found in lookup. Then result holds the value associated with key.
// If the Handler returns a non-nil err this error gets passed back to the client.
// You can use [PermanentError] or [TimeoutError] to signal a permanent or timeout error to the client.
// All other errors are returned as temporary errors.
type Handler func(ctx context.Context, lookup, key string) (result string, found bool, err error)

// Serve accepts incoming socketmap connections on the listener l, creating a new service goroutine for each.
// The service goroutines read requests and then call handler to reply to them.
//
// Serve always returns a non-nil error.
func Serve(l net.Listener, handler Handler) error {
	if handler == nil {
		panic("handler cannot be nil")
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go func() {
			_ = handle(conn, handler)
		}()
	}
}

// ListenAndServe listens on the network address addr and then calls Serve with handler to handle requests on incoming connections.
//
// ListenAndServe always returns a non-nil error.
func ListenAndServe(network, addr string, handler Handler) error {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	return Serve(ln, handler)
}

func handle(conn net.Conn, handler Handler) error {
	for {
		b, err := read(conn)
		if err != nil {
			return conn.Close()
		}
		parts := strings.SplitN(string(b), " ", 2)
		if len(parts) != 2 {
			return conn.Close()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		result, found, err := handler(ctx, parts[0], parts[1])
		select {
		case <-ctx.Done():
			err = write(conn, []byte("TIMEOUT "), []byte(ctx.Err().Error()))
			cancel()
			if err != nil {
				return conn.Close()
			}
		default:
			cancel()
		}
		if err != nil {
			switch err.(type) {
			case PermanentError, *PermanentError:
				err = write(conn, []byte("PERM "), []byte(err.Error()))
			case TimeoutError, *TimeoutError:
				err = write(conn, []byte("TIMEOUT "), []byte(err.Error()))
			default:
				err = write(conn, []byte("TEMP "), []byte(err.Error()))
			}
		} else {
			if found {
				err = write(conn, []byte("OK "), []byte(result))
			} else {
				err = write(conn, []byte("NOTFOUND"))
			}
		}
		if err != nil {
			return conn.Close()
		}
	}
}
