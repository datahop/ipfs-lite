package repo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	root := filepath.Join("../../test", "root1")
	defer func() {
		err := os.RemoveAll(root)
		if err != nil {
			t.Fatal(err)
		}
	}()
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	cfgFName, err := ConfigFilename(root)
	_, err = os.Stat(cfgFName)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpen(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../../test", "root1")
	defer func() {
		err := os.RemoveAll(root)
		if err != nil {
			t.Fatal(err)
		}
	}()
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	_, err = os.Stat(filepath.Join(root, DefaultDatastoreFolderName))
	if err != nil {
		t.Fatal(err)
	}
}
