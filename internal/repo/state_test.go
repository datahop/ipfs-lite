package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStateKeeper(t *testing.T) {
	root := filepath.Join("./test", "root1")
	err := os.MkdirAll(root, 0777)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadStateKeeper(root)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo("./test", t)
}

func TestLoadStateKeeperTwice(t *testing.T) {
	root := filepath.Join("./test", "root1")
	err := os.MkdirAll(root, 0777)
	if err != nil {
		t.Fatal(err)
	}
	s, err := LoadStateKeeper(root)
	if err != nil {
		t.Fatal(err)
	}
	defer removeRepo("./test", t)

	_, err = s.AddOrUpdateState("myState", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		_, err = s.AddOrUpdateState(fmt.Sprintf("%d", i), true, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	s2, err := LoadStateKeeper(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(s2.states) != 10 {
		t.Fatal("myState is not being loaded")
	}
}
