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
	_, err = os.Stat(filepath.Join(root, DefaultStateFile))
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

func TestStateWithNoStateFile(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../../test", "root1")
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	e := os.Remove(filepath.Join(root, DefaultStateFile))
	if e != nil {
		log.Fatal(e)
	}
	r, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if r.State() != 0 {
		t.Fatal("initial state should be 0")
	}
}

func TestStateWithUnsupportedValue(t *testing.T) {
	<-time.After(time.Second)
	root := filepath.Join("../../test", "root1")
	err := Init(root, "0")
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo(root, t)
	f, err := os.Create(filepath.Join(root, DefaultStateFile))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("UnsupportedValue")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	r, err := Open(root)
	if err == nil {
		r.Close()
		t.Fatal(err)
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
	defer r.Close()
	if r.State() != 0 {
		t.Fatal("initial state should be 0")
	}
	err = r.SetState(100)
	if err != nil {
		t.Fatal(err)
	}
	if r.State() != 100 {
		t.Fatal("initial state should be 100")
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
		t.Fatal(err)
	}
}

func removeRepo(repopath string, t *testing.T) {
	err := os.RemoveAll(repopath)
	if err != nil {
		t.Fatal(err)
	}
}
