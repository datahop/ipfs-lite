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

type State struct {
	Filter     *bloom.BloomFilter
	Membership bool
}

type StateKeeper struct {
	states map[string]*State
	root   string
	mtx    sync.Mutex
}

func LoadStateKeeper(root string) (*StateKeeper, error) {
	sk := &StateKeeper{
		states: map[string]*State{},
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
	d := map[string]*State{}
	err = json.Unmarshal(data, &d)
	if err != nil {
		return nil, err
	}
	sk.states = d
	return sk, nil
}

func (s *StateKeeper) GetStates() map[string]*State {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.states
}

func (s *StateKeeper) SaveStates() (map[string]*State, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.saveStates()
}

func (s *StateKeeper) saveStates() (map[string]*State, error) {
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

func (s *StateKeeper) AddOrUpdateState(name string, membership bool, deltas []string) (*bloom.BloomFilter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.states[name] == nil {
		s.states[name] = &State{
			Filter: bloom.New(uint(2000), 5),
		}
	}
	if len(deltas) > 0 {
		for _, v := range deltas {
			s.states[name].Filter = s.states[name].Filter.AddString(v)
		}
	}
	s.states[name].Membership = membership
	_, err := s.saveStates()
	if err != nil {
		return nil, err
	}
	return s.states[name].Filter, nil
}

func (s *StateKeeper) GetState(name string) (*State, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.states[name] == nil {
		return nil, fmt.Errorf("no state available")
	}
	return s.states[name], nil
}
