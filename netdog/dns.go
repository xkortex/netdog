package netdog

import (
	"fmt"
	"golang.org/x/net/context"
	"net"
)

type AddrResponse struct {
	addrs []string
	err   error
}

func TimeoutLookupHost(host string, timeout float64) (addrs []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), SecsToDuration(timeout))
	defer cancel()

	ch := make(chan AddrResponse, 1)
	defer close(ch)

	go func() {
		select {
		default:
			addrs, err := net.LookupHost(host)
			ch <- AddrResponse{addrs: addrs, err: err}
		case <-ctx.Done():
			fmt.Println("Exiting gofunc")
			return
		}
	}()

	select {
	case out := <-ch:
		return out.addrs, out.err
	case <-ctx.Done():
		return nil, fmt.Errorf("Lookup '%s' timed out after %.3fs", host, timeout)
	}
}
