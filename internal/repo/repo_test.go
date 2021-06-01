package repo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	root := filepath.Join("../../test", "root1")
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
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
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
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

func TestConfigError(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../../test", "root1")
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	_, err = r.Config()
	if err != ErrorRepoClosed {
		t.Fatal(err)
	}
}

func removeRepo(repopath string, t *testing.T) {
	err := os.RemoveAll(repopath)
	if err != nil {
		t.Fatal(err)
	}
}
