package repo

import (
	"bytes"
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
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(cfgFName)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(filepath.Join(root, defaultStateFile))
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
	_, err = os.Stat(filepath.Join(root, defaultDatastoreFolderName))
	if err != nil {
		t.Fatal(err)
	}
}

func TestStateWithNoStateFile(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../../test", "root1")
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	e := os.Remove(filepath.Join(root, defaultStateFile))
	if e != nil {
		log.Fatal(e)
	}
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if r.State() == nil {
		t.Fatal("initial state should not be nil")
	}
}

func TestStateValues(t *testing.T) {
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
	if r.State() == nil {
		r.Close()
		t.Fatal("initial state should not be nil")
	}

	bf1, err := r.State().MarshalJSON()
	if err != nil {
		r.Close()
		t.Fatal(err)
	}
	r.Close()
	r, err = Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	bf2, err := r.State().MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bf1, bf2) {
		t.Fatal("bloom filters should be identical")
	}
	newBloom := r.State().Add([]byte("This is a text"))
	bf3, err := r.State().MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	bf4, err := newBloom.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !newBloom.Equal(r.State()) {
		t.Fatal("bloom filters should be identical")
	}
	if !bytes.Equal(bf3, bf4) {
		t.Fatal("bloom filters should be identical")
	}
}

func TestConfig(t *testing.T) {
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
	_, err = r.Config()
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

func TestDatastore(t *testing.T) {
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
	ds := r.Datastore()
	if ds == nil {
		t.Fatal("datastore is null")
	}
}

func removeRepo(repopath string, t *testing.T) {
	err := os.RemoveAll(repopath)
	if err != nil {
		t.Fatal(err)
	}
}
