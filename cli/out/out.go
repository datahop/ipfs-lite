package out

import (
	"bytes"
	"encoding/json"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("out")

type Printer interface {
	Println(...interface{})
}

// Format of the printer
type Format int

const (
	// PrettyJson prints indented json
	PrettyJson Format = iota
	Json
	NoStyle
)

// Print data
func Print(p Printer, data interface{}, f Format) error {
	switch f {
	case PrettyJson:
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, b, "", "\t")
		if err != nil {
			return err
		}
		p.Println(prettyJSON.String())
		return nil
	case Json:
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		p.Println(string(b))
	default:
		p.Println(data)
	}
	return nil
}
