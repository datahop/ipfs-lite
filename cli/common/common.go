package common

import "context"

type Common struct {
	IsDaemon bool
	Context  context.Context
	Cancel   context.CancelFunc
}
