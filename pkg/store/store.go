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
	GroupAdd(context.Context, io.Reader, *Info, string) (string, error)
	AddDir(context.Context, string, *Info) (string, error)
	GroupAddDir(context.Context, string, *Info, string) (string, error)
	Get(context.Context, string) (io.ReadSeekCloser, *Info, error)
	Delete(context.Context, string) error
}
