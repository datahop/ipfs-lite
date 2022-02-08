package store

import (
	"context"
	"io"
)

type Info struct {
	Tag         string
	Type        string
	Name        string
	IsEncrypted bool
	Size        int64
}

type Store interface {
	Add(context.Context, io.Reader, *Info) (string, error)
	AddDir(context.Context, string, *Info) (string, error)
	Get(context.Context, string) (io.ReadSeekCloser, *Info, error)
	Delete(context.Context, string) error
}
