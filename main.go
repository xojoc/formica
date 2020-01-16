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

package main // import "xojoc.pw/formica"

import (
	"bufio"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"text/template"
)

type item struct {
	Id              int
	Title           string
	Excerpt         string
	Slug            string
	Date            interface{} // go-yaml is buggy, so we must deal with dates ourselfs
	Year            int
	Month           int
	Day             int
	Tags            []string
	GoPath          string
	GoCode          string
	GoDocumentation string
	User            map[string]interface{} // user variables

	inpath  string
	outpath string

	src io.ReadCloser
	buf *bufio.Reader

	r *rule // FIXME: refactor collect.go and remove this field
}

type rule struct {
	In           string
	inre         *regexp.Regexp
	Out          string
	outtpl       *template.Template
	Exec         string
	copy         bool
	NoHeader     bool
	Dependencies []string //FIXME: should be a list

	s *section
}

type section struct {
	URL        string
	Dir        string
	Rules      []*rule
	Title      string
	Excerpt    string
	Style      string
	IncludeCSS []string
	IncludeJS  []string
	IndexSort  string
	Feed       bool

	items []*item
}

var AllSections []*section

var Config struct {
	SiteURL string
}

const buildDir = "_build/"

func main() {
	log.SetPrefix(path.Base(os.Args[0]) + ": ")
	log.SetFlags(log.Lshortfile)
	parseConfig()
	collectItems()
	renderAll()
	copyAssets()
}
