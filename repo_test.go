package ipfslite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	root := filepath.Join("./test", "root")
	conf, err := ConfigInit(2048, "0")
	if err != nil {
		t.Fatal(err)
	}
	err = Init(root, conf)
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
	root := filepath.Join("./test", "root")
	conf, err := ConfigInit(2048, "0")
	if err != nil {
		t.Fatal(err)
	}
	err = Init(root, conf)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Datastore().Close()
	_, err = os.Stat(filepath.Join(root, DefaultDatastoreFolderName))
	if err != nil {
		t.Fatal(err)
	}
}
