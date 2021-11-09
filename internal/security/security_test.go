package security

import (
	"bytes"
	"testing"
)

func TestNew(t *testing.T) {
	text := []byte("test text")
	pass := "pass"
	b1, err := Encrypt(text, pass)
	if err != nil {
		t.Fatal("encrypt failed ", err.Error())
	}

	newText, err := Decrypt(b1, pass)
	if err != nil {
		t.Fatal("encrypt failed ", err.Error())
	}
	if !bytes.Equal(newText, text) {
		t.Fatal("result does not match")
	}
}
