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
	"bufio"
	"bytes"
	yaml "gopkg.in/yaml.v2"
	htpl "html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func openBuf(i *item) {
	if i.src != nil {
		log.Fatalf("trying to reopen %q\n", i.inpath)
	}
	f, err := os.Open(i.inpath)
	if err != nil {
		log.Fatal(err)
	}
	i.src = f
	i.buf = bufio.NewReader(i.src)
}

func closeBuf(i *item) {
	if i.src == nil {
		log.Fatalf("trying to reclose %q\n", i.inpath)
	}
	err := i.src.Close()
	if err != nil {
		log.Fatal(err)
	}
	i.src = nil
	i.buf = nil
}

// FIXME: use buffer inside itemContext
func GetBody(i *item) htpl.HTML {
	defer closeBuf(i)
	if i.src == nil {
		openBuf(i)
	}
	var buf bytes.Buffer
	execShellIO(i.r.Exec, i.buf, &buf, nil)
	return htpl.HTML(buf.String())
}

func metaFromPath(i *item) {
	expand := func(field string) string {
		return i.r.inre.ReplaceAllString(i.inpath, "${"+field+"}")
	}

	expandInt := func(field string) (bool, int) {
		t := expand(field)
		if t != "" {
			i, err := strconv.ParseInt(t, 10, 32)
			if err != nil {
				log.Fatal(err)
			}
			return true, int(i)
		} else {
			return false, 0
		}
	}

	for _, name := range i.r.inre.SubexpNames() {
		if name == "" {
			continue
		}
		switch name {
		case "id":
			if ok, id := expandInt("id"); ok {
				i.Id = id
			}
		case "slug":
			slug := expand("slug")
			if slug != "" {
				i.Slug = slug
			}
		case "year":
			if ok, y := expandInt("year"); ok {
				i.Year = int(y)
			}
		case "month":
			if ok, m := expandInt("month"); ok {
				i.Month = int(m)
			}
		case "day":
			if ok, d := expandInt("day"); ok {
				i.Day = int(d)
			}
		default: /* user value */
			if ok, n := expandInt(name); ok {
				i.User[name] = n
			} else {
				i.User[name] = expand(name)
			}
		}
	}
}

func getHeader(i *item) []byte {
	openBuf(i)
	h := make([]byte, 0)
	for {
		l, err := i.buf.ReadBytes('\n')
		if err != nil {
			log.Fatal(err)
		}
		if err == io.EOF {
			log.Fatalf("%q: got EOF while parsing header", i.inpath)
		}
		if string(l) == "...\n" {
			return h
		}
		h = append(h, l...)
	}
}

func metaFromHeader(i *item) {
	if i.r.NoHeader {
		return
	}
	h := getHeader(i)
	if h == nil {
		return
	}
	err := yaml.Unmarshal(h, i)
	if err != nil {
		log.Fatalf("%s: %v", i.inpath, err)
	}
}

func metaInfer(i *item) {
	if i.Date != nil {
		var err error
		i.Date, err = time.Parse("2006-01-02", i.Date.(string))
		if err != nil {
			log.Fatal(err)
		}

		i.Year = i.Date.(time.Time).Year()
		i.Month = int(i.Date.(time.Time).Month())
		i.Day = i.Date.(time.Time).Day()
		/*

			if i.Date.(time.Time).IsZero() && (i.Year != 0 && int(i.Month) != 0 && i.Day != 0) {
				i.Date = time.Date(i.Year, time.Month(i.Month), i.Day, 0, 0, 0, 0, time.UTC)
			} else if !i.Date.(time.Time).IsZero() {
				i.Year = i.Date.(time.Time).Year()
				i.Month = int(i.Date.(time.Time).Month())
				i.Day = i.Date.(time.Time).Day()
			}
		*/
	} else {
		if i.Year != 0 && i.Month != 0 && i.Day != 0 {
			i.Date = time.Date(i.Year, time.Month(i.Month), i.Day, 0, 0, 0, 0, time.UTC)
		}
	}
}

func fileToItem(f string, r *rule) *item {
	i := &item{}
	i.Id = -1
	i.inpath = f
	i.r = r
	i.User = make(map[string]interface{})
	metaFromPath(i)
	metaFromHeader(i)
	metaInfer(i)

	// FIXME
	if i.r.NoHeader == false {
		if i.Title == "" {
			log.Fatalf("item %q has no `Title`", i.inpath)
		}
		if i.Slug == "" {
			log.Fatalf("item %q has no `Slug`", i.inpath)
		}
		if i.Id < 0 {
			log.Printf("item %q has no `Id`", i.inpath)
		}
	}

	// FIXME: maybe remove
	/*
		if i.Date == nil {
			log.Printf("item %q has no `Date`", i.inpath)
		}
	*/
	if i.r.Out == "" {
		i.outpath = buildDir + i.inpath
	} else {
		var b bytes.Buffer
		err := i.r.outtpl.Execute(&b, i)
		if err != nil {
			log.Fatal(err)
		}
		i.outpath = buildDir + b.String()
	}

	return i
}

func collectItem(f string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	for _, s := range AllSections {
		for _, r := range s.Rules {
			if s.Dir == "." && strings.IndexByte(f, '/') == -1 {
				f = "./" + f
			}
			if r.inre.MatchString(f) {
				i := fileToItem(f, r)
				if r.copy {
					copyFile(i.inpath, i.outpath)
				} else {
					s.items = append(s.items, i)
				}
				return nil
			}
		}
		/*
			// Ignore dot files
			if filepath.Base(f)[0] != '.' {
				if strings.HasPrefix(f, section.Dir+"/") || (section.Dir == "." && strings.IndexByte(f, '/') == -1) {
					copyFile(f, buildDir+f)
					return nil
				}
			}
		*/
	}
	return nil
}

func collectItems() {
	err := filepath.Walk(".", collectItem)
	if err != nil {
		log.Fatal(err)
	}
}
