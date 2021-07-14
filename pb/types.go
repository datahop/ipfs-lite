package types

//go:generate protoc -I. --go_out=. types.proto
//go:generate protoc -I. --java_out=. types.proto

import "fmt"

// Reference imports to suppress errors
var _ = fmt.Errorf
