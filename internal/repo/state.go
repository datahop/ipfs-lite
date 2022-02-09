package repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

const (
	stateKeeperFile = ".statekeeper"
)

type StateKeeper struct {
	states map[string]*bloom.BloomFilter
	root   string
	mtx    sync.Mutex
}

func loadStateKeeper(root string) (*StateKeeper, error) {
	sk := &StateKeeper{
		states: map[string]*bloom.BloomFilter{},
		mtx:    sync.Mutex{},
		root:   root,
	}
	if _, err := os.Stat(filepath.Join(root, stateKeeperFile)); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		_, err = os.Create(filepath.Join(root, stateKeeperFile))
		if err != nil {
			return nil, err
		}
		d, err := json.Marshal(sk.states)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(filepath.Join(root, stateKeeperFile), d, 0664)
		if err != nil {
			return nil, err
		}
		return sk, nil
	}
	data, err := ioutil.ReadFile(filepath.Join(root, stateKeeperFile))
	if err != nil {
		return nil, err
	}
	d := map[string]*bloom.BloomFilter{}
	err = json.Unmarshal(data, &d)
	if err != nil {
		return nil, err
	}
	sk.states = d
	return sk, nil
}

func (s *StateKeeper) GetStates() map[string]*bloom.BloomFilter {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.states
}

func (s *StateKeeper) SaveStates() (map[string]*bloom.BloomFilter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.saveStates()
}

func (s *StateKeeper) saveStates() (map[string]*bloom.BloomFilter, error) {
	d, err := json.Marshal(s.states)
	if err != nil {
		return s.states, err
	}
	err = ioutil.WriteFile(filepath.Join(s.root, stateKeeperFile), d, 0664)
	if err != nil {
		return s.states, err
	}
	return s.states, nil
}

func (s *StateKeeper) AddNewStates(name string) (map[string]*bloom.BloomFilter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.states[name] = bloom.New(uint(2000), 5)
	return s.saveStates()
}

func (s *StateKeeper) UpdateStates(name string, delta []byte) (*bloom.BloomFilter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.states[name] == nil {
		return nil, fmt.Errorf("state not available")
	}
	s.states[name] = s.states[name].Add(delta)
	_, err := s.saveStates()
	if err != nil {
		return nil, err
	}
	return s.states[name], nil
}

func (s *StateKeeper) GetState(name string) *bloom.BloomFilter {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.states[name]
}
