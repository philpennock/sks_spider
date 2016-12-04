/*
   Copyright 2016 Phil Pennock

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package sks_spider

// sorted set of strings:
//go:generate gengen -o ./internal/string_set github.com/joeshaw/gengen/examples/btree string struct{}

import (
	"strings"

	btree "github.com/philpennock/sks_spider/internal/string_set"
)

type sortedSet struct {
	*btree.Tree
}

func newSortedSet() sortedSet {
	return sortedSet{
		Tree: btree.TreeNew(strings.Compare),
	}
}

// eww, code smell XXX
func hostCompare(a, b string) int {
	a1 := strings.Split(a, ".")
	b1 := strings.Split(b, ".")
	ReverseStringSlice(a1)
	ReverseStringSlice(b1)
	a2 := strings.Join(a1, ".")
	b2 := strings.Join(b1, ".")
	return strings.Compare(a2, b2)
}

func newHostReversedSet() sortedSet {
	return sortedSet{
		Tree: btree.TreeNew(hostCompare),
	}
}

func (set sortedSet) Insert(key string) {
	set.Tree.Set(key, struct{}{})
}

func (set sortedSet) Contains(key string) bool {
	_, ok := set.Tree.Seek(key)
	return ok
}

func (set sortedSet) Len() int {
	return set.Tree.Len()
}

func (set sortedSet) Remove(key string) {
	_ = set.Tree.Delete(key)
}

func (set sortedSet) Data() <-chan string {
	ch := make(chan string, 100)
	go func(c chan<- string) {
		enum, err := set.Tree.SeekFirst()
		if err != nil {
			close(c)
			return
		}
		for {
			k, _, err := enum.Next()
			if err != nil {
				break
			}
			c <- k
		}
		close(c)
		// enum.Close() -- in github.com/cznic/b docs but not in template
		return
	}(ch)
	return ch
}

func (set sortedSet) AllData() []string {
	if set.Tree.Len() == 0 {
		return nil
	}
	dl := make([]string, 0, set.Tree.Len())
	enum, err := set.Tree.SeekFirst()
	if err != nil {
		return dl
	}
	for {
		k, _, err := enum.Next()
		if err != nil {
			break
		}
		dl = append(dl, k)
	}
	// enum.Close() -- in github.com/cznic/b docs but not in template
	return dl
}

type collectionOfSortedSets map[string]sortedSet

func newCollectionOfSortedSets(size int) collectionOfSortedSets {
	return make(collectionOfSortedSets, size)
}

func (css collectionOfSortedSets) Ensure(key string) {
	if _, ok := css[key]; !ok {
		css[key] = newSortedSet()
	}
}

func (css collectionOfSortedSets) Insert(key, setKey string) {
	css.Ensure(key)
	css[key].Insert(setKey)
}
