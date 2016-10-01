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
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"text/template"
)

const (
	cfgDir  = "_formica"
	cfgName = "config.yaml"
)

func pathToRe(in, dir string) *regexp.Regexp {
	in = strings.NewReplacer(
		`{id}`, `(?P<id>[[:digit:]]+)`,
		`{title}`, `(?P<title>[^/]+)`,
		`{slug}`, `(?P<slug>[^/]+)`,
		`{year}`, `(?P<year>[[:digit:]]{4})`,
		`{month}`, `(?P<month>[[:digit:]]{2})`,
		`{day}`, `(?P<day>[[:digit:]]{2})`).Replace(in)
	restr := "^" + regexp.QuoteMeta(dir) + in + "$"
	re, err := regexp.Compile(restr)
	if err != nil {
		log.Fatalf("%q: %s", restr, err.Error())
	}

	return re
}

func pathToTpl(out, dir string) *template.Template {
	out = strings.NewReplacer(
		`{id}`, `{{.Id}}`,
		`{title}`, `{{.Title}}`,
		`{slug}`, `{{.Slug}}`,
		`{year}`, `{{.Year}}`,
		`{month}`, `{{.Month}}`,
		`{day}`, `{{.Day}}`).Replace(out)
	re := regexp.MustCompile(`{user\.(.+)}`)
	out = re.ReplaceAllString(out, `{{.User.$1}}`)
	tpl, err := template.New("").Parse(dir + out)
	if err != nil {
		log.Fatalf("%q: %s", dir+out, err.Error())
	}

	return tpl
}

func parseConfig() {
	buf, err := ioutil.ReadFile(cfgDir + "/" + cfgName)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(buf, &AllSections)
	if err != nil {
		log.Fatal(err)
	}
	// Check mandatory fields and set defaults.
	for si, s := range AllSections {
		if s.Dir == "" {
			log.Fatalf("no `dir` specified for section n. %d\n", si+1)
		}
		if s.Rules == nil {
			log.Fatalf("no `rules` specified for section n. %d (%q)\n", si+1, s.Dir)
		}
		if s.Title == "" {
			//			log.Printf("no `title` specified for section n. %d (%q)\n", si+1, s.Dir)
			s.Title = "(no title)"
		}
		if s.Style == "" {
			s.Style = styleDef
		}
		for ri, r := range s.Rules {
			r.s = s

			if r.In == "" {
				log.Fatalf("no `In` filter specified for rule n. %d in section n. %d (%q)", ri+1, si+1, s.Dir)
			}
			if r.Out == "" && r.Exec != "" {
				log.Printf("no `Out` filter specified for rule n. %d (section n. %d %q)", ri+1, si+1, s.Dir)
			}
			if r.Exec == "" {
				r.copy = true
				r.NoHeader = true
				//				log.Fatalf("no `Exec` specified for rule n. %d in section n. %d (%q)", ri+1, si+1, s.Dir)
			}

			var dir string
			// little kludge
			//			if s.Dir == "." {
			//				dir = ""
			//			} else {
			dir = s.Dir + "/"
			//			}
			r.inre = pathToRe(r.In, dir)
			r.outtpl = pathToTpl(r.Out, dir)
		}
	}
}
