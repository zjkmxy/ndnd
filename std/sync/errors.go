package sync

import (
	"errors"
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
)

var ErrSnapshot = errors.New("snapshot error")

type ErrSync struct {
	name enc.Name
	err  error
}

func (e *ErrSync) Name() enc.Name {
	return e.name
}

func (e *ErrSync) Error() string {
	return fmt.Sprintf("sync error [%s]: %v", e.name, e.err)
}

func (e *ErrSync) Unwrap() error {
	return e.err
}
