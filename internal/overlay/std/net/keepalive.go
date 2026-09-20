//go:build dart

package net

import (
	"syscall"
	"time"
)

// KeepAliveConfig controls TCP keepalive probes. Zero values select defaults.
// Negative Idle, Interval, and Count values preserve the socket options.
type KeepAliveConfig struct {
	Enable   bool
	Idle     time.Duration
	Interval time.Duration
	Count    int
}

func (c *TCPConn) SetKeepAlivePeriod(d time.Duration) error {
	if c.c.network == "" {
		return syscall.EINVAL
	}
	return c.keepAliveDuration("keepidle", d)
}

func (c *TCPConn) keepAliveDuration(name string, d time.Duration) error {
	if d < 0 {
		return nil
	}
	if d == 0 {
		d = 15 * time.Second
	}
	return c.c.setOption(name, int((d+time.Second-1)/time.Second))
}

func (c *TCPConn) SetKeepAliveConfig(config KeepAliveConfig) error {
	if c.c.network == "" {
		return syscall.EINVAL
	}
	if err := c.SetKeepAlive(config.Enable); err != nil {
		return err
	}
	if err := c.keepAliveDuration("keepidle", config.Idle); err != nil {
		return err
	}
	if err := c.keepAliveDuration("keepinterval", config.Interval); err != nil {
		return err
	}
	if config.Count < 0 {
		return nil
	}
	if config.Count == 0 {
		config.Count = 9
	}
	return c.c.setOption("keepcount", config.Count)
}
