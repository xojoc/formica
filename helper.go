/* Copyright (C) 2014, 2015 by Alexandru Cojocaru */

/* This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>. */

package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func execShellIO(cmdstr string, in io.Reader, out io.Writer, stderr io.Writer) {
	cmd := exec.Command("sh", "-c", cmdstr)
	cmd.Stdin = in
	cmd.Stdout = out
	if stderr != nil {
		cmd.Stderr = stderr
	} else {
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err != nil {
		log.Fatalf("%q: %v\n", cmdstr, err)
	}
}
func execShell(cmdstr string) {
	execShellIO(cmdstr, nil, nil, nil)
}

func copyFile(i string, o string) {
	if !newer(i, o) {
		return
	}
	err := os.MkdirAll(filepath.Dir(o), 0755)
	if err != nil {
		log.Fatal(err)
	}
	execShell("cp " + i + " " + o)
}

func newer(a string, b string) bool {
	sta, err := os.Stat(a)
	if err != nil {
		log.Fatal(err) // FIXME: maybe return false
	}
	stb, err := os.Stat(b)
	if err != nil {
		return true
	}
	return sta.ModTime().After(stb.ModTime())
}

func newerGlob(glob string, b string) bool {
	fs, err := filepath.Glob(glob)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range fs {
		if newer(f, b) {
			return true
		}
	}
	return false
}

// mmmh pretty ugly but works

type lessFunc func(i1, i2 *item) int

type multiSorter struct {
	items []*item
	less  []lessFunc
}

func (ms multiSorter) Sort(items []*item) {
	ms.items = items
	sort.Sort(ms)
}

var _ sort.Interface = multiSorter{}

func SortBy(items []*item, less ...lessFunc) multiSorter {
	return multiSorter{
		items: items,
		less:  less,
	}
}

func (ms multiSorter) Len() int {
	return len(ms.items)
}
func (ms multiSorter) Swap(i, j int) {
	ms.items[i], ms.items[j] = ms.items[j], ms.items[i]
}

func (ms multiSorter) Less(i, j int) bool {
	i1, i2 := ms.items[i], ms.items[j]
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		cmp := less(i1, i2)
		switch {
		case cmp < 0:
			return true
		case cmp > 0:
			return false
		}
	}
	return ms.less[k](i1, i2) <= 0
}

func intCmp(_i, _j interface{}) int {
	i := _i.(int)
	j := _j.(int)
	if i < j {
		return -1
	} else if i > j {
		return 1
	} else {
		return 0
	}
}
func stringCmp(_i, _j interface{}) int {
	i := _i.(string)
	j := _j.(string)
	if i < j {
		return -1
	} else if i > j {
		return 1
	} else {
		return 0
	}
}
func timeCmp(_i, _j interface{}) int {
	i := _i.(time.Time)
	j := _j.(time.Time)
	if i.Before(j) {
		return -1
	} else if i.After(j) {
		return 1
	} else {
		return 0
	}
}

func idCmp(i1, i2 *item) int {
	return intCmp(i1.Id, i2.Id)
}
func titleCmp(i1, i2 *item) int {
	return stringCmp(i1.Title, i2.Title)
}
func dateCmp(i1, i2 *item) int {
	return timeCmp(i1.Date, i2.Date)
}

func userCmp(key string) lessFunc {
	return func(i1, i2 *item) int {
		var cmp func(i, j interface{}) int
		if i1.User[key] == nil {
			log.Printf("no value for key %q for item %q\n", key, i1.inpath)
			return 1
		}
		if i2.User[key] == nil {
			log.Printf("no value for key %q for item %q\n", key, i2.inpath)
			return 1
		}

		switch i1.User[key].(type) {
		case int:
			cmp = intCmp
		case string:
			cmp = stringCmp
		default:
			log.Fatalf("type %T not handled (please report it)\n", key)
		}
		return cmp(i1.User[key], i2.User[key])
	}
}

func SortItemsBy(items []*item, keys ...string) []*item {
	reverse := false
	less := make([]lessFunc, 0)
	for _, k := range keys {
		if strings.HasPrefix(k, "-") {
			reverse = true
			k = k[1:]
		}
		switch k {
		case "id":
			less = append(less, idCmp)
		case "title":
			less = append(less, titleCmp)
		case "date":
			less = append(less, dateCmp)
		default:
			less = append(less, userCmp(k))
		}
	}
	var s sort.Interface = SortBy(items, less...)
	if reverse {
		s = sort.Reverse(s)
	}
	sort.Sort(s)
	return items
}
