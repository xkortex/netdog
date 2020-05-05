package netdog

import "time"

func SecsToDuration(t float64) time.Duration {
	timeout_ns := t * 1e9
	return time.Duration(int(timeout_ns))
}
