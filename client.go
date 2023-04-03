package socketmap

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/puddle/v2"
)

type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type clientOptions struct {
	dialer  Dialer
	maxSize int32
}

type ClientOption func(clientOptions)

func WithDialer(dialer Dialer) ClientOption {
	return func(options clientOptions) {
		options.dialer = dialer
	}
}

func WithMaxSize(maxSize int32) ClientOption {
	return func(options clientOptions) {
		options.maxSize = maxSize
	}
}

type Client struct {
	p *puddle.Pool[net.Conn]
}

// NewClient returns a new [Client] that connects to a socketmap server with network and address.
// You can pass additional opts. The defaults are using maximum of 10 parallel connections and a
// net.Dialer with a 10 seconds timeout.
func NewClient(network, addr string, opts ...ClientOption) *Client {
	o := clientOptions{maxSize: 10}
	for _, f := range opts {
		f(o)
	}
	if o.dialer == nil {
		o.dialer = &net.Dialer{Timeout: 10}
	}
	p, err := puddle.NewPool[net.Conn](&puddle.Config[net.Conn]{
		Constructor: func(ctx context.Context) (res net.Conn, err error) {
			return o.dialer.DialContext(ctx, network, addr)
		},
		Destructor: func(res net.Conn) {
			_ = res.Close()
		},
		MaxSize: o.maxSize,
	})
	if err != nil {
		panic(err)
	}
	return &Client{p: p}
}

// Lookup calls LookupContext with [context.Background] as ctx.
func (c *Client) Lookup(lookup, key string) (string, bool, error) {
	return c.LookupContext(context.Background(), lookup, key)
}

// LookupContext lookup key in the lookup map of c.
// If ctx is Done while looking up key, err will be ctx.Err().
// If the map has a value, ok will be true. If map does not have a value for key, value will be "" and ok will be false.
// If there is a communication error or the socketmap server returns an error, err will be non-nil.
func (c *Client) LookupContext(ctx context.Context, lookup, key string) (value string, ok bool, err error) {
	if ctx.Done() != nil {
		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		default:
		}
	}
	res, err := c.p.Acquire(ctx)
	if err != nil {
		return "", false, err
	}
	if ctx.Done() != nil {
		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		default:
		}
	}
	chunks := [][]byte{[]byte(lookup), {' '}, []byte(key)}
	err = write(res.Value(), chunks...)
	// retry write once with a new connection on error if connection was stale
	if err != nil {
		res.Destroy()
		if ctx.Done() != nil {
			select {
			case <-ctx.Done():
				return "", false, ctx.Err()
			default:
			}
		}
		res, err = c.p.Acquire(ctx)
		if err != nil {
			return "", false, err
		}
		if ctx.Done() != nil {
			select {
			case <-ctx.Done():
				return "", false, ctx.Err()
			default:
			}
		}
		err = write(res.Value(), chunks...)
	}
	defer res.Release()
	if err != nil {
		return "", false, err
	}
	if ctx.Done() != nil {
		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		default:
		}
	}
	b, err := read(res.Value())
	if err != nil {
		return "", false, err
	}
	parts := strings.SplitN(string(b), " ", 2)
	if len(parts) != 2 || parts[0] != "NOTFOUND" {
		return "", false, fmt.Errorf("unknown response %q", b)
	}
	switch parts[0] {
	case "OK":
		return parts[1], true, nil
	case "NOTFOUND":
		return "", false, nil
	case "TIMEOUT":
		return "", false, &TimeoutError{Reason: parts[1]}
	case "TEMP":
		return "", false, &TempError{Reason: parts[1]}
	case "PERM":
		return "", false, &PermanentError{Reason: parts[1]}
	default:
		return "", false, &TempError{Reason: fmt.Sprintf("unknown response %q", b)}
	}
}

// Close finishes all connections. Waits until all connections are closed.
// After the call to Close all LookupContext calls will return an error.
func (c *Client) Close() {
	c.p.Close()
}
