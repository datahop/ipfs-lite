package security

import (
	"bytes"
	"testing"
)

func TestNew(t *testing.T) {
	text := []byte("test text")
	pass := "pass"
	d := &DefaultEncryption{}
	b1, err := d.EncryptContent(text, pass)
	if err != nil {
		t.Fatal("encrypt failed ", err.Error())
	}

	newText, err := d.DecryptContent(b1, pass)
	if err != nil {
		t.Fatal("encrypt failed ", err.Error())
	}
	if !bytes.Equal(newText, text) {
		t.Fatal("result does not match")
	}
}
