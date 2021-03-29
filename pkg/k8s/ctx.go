package k8s

import (
	"time"

	errors2 "github.com/pkg/errors"
)

const (
	CtxTimeout = 10 * time.Second
)

func TimeoutWrap(timeout time.Duration, fn func() error) error {
	result := make(chan error)
	go func() {
		result <- fn()
	}()

	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		return errors2.New("timeout")
	}
}
