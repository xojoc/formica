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
	"bytes"
	"fmt"
	htpl "html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const baseDir = "/"

type sectionContext struct {
	Dir   string
	Title string
	Tags  []*tagContext
	Items []*itemContext

	section *section
}

func contextFromSection(s *section) *sectionContext {
	sctx := &sectionContext{}
	sctx.Dir = s.Dir
	sctx.Title = s.Title
	var is []*itemContext
	for _, i := range s.items {
		is = append(is, contextFromItem(i, sctx))
	}
	sctx.Items = is
	sctx.section = s
	return sctx
}
func (s *sectionContext) PageTitle() string {
	return s.Title
}
func (s *sectionContext) AbsoluteURL() string {
	return baseDir + s.Dir
}
func (s *sectionContext) HomeURL() string {
	return s.AbsoluteURL()
}
func (s *sectionContext) RootURL() string {
	return baseDir
}
func (s *sectionContext) HomeTitle() string {
	return s.Title
}
func (s *sectionContext) Include() htpl.HTML {
	fs, err := filepath.Glob(stylePath(s.section.Style) + "*.css")
	if err != nil {
		log.Fatal(err)
	}
	str := ""
	for _, f := range fs {
		str += fmt.Sprintf(`<link rel="stylesheet" href="%s" type="text/css">`, baseDir+"css/"+s.section.Style+"/"+filepath.Base(f))
	}
	for _, i := range s.section.IncludeCSS {
		str += fmt.Sprintf(`<link rel="stylesheet" href="%s" type="text/css">`, i)
	}

	fs, err = filepath.Glob(stylePath(s.section.Style) + "*.js")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range fs {
		str += fmt.Sprintf(`<script type="text/javascript" src="%s"></script>`, baseDir+"js/"+s.section.Style+"/"+filepath.Base(f))
	}

	for _, i := range s.section.IncludeJS {
		str += fmt.Sprintf(`<script type="text/javascript" src="%s"></script>`, i)
	}

	return htpl.HTML(str)
}

type tagContext struct {
	Tag   string
	Items []*itemContext
}

func contextFromTag(tag string, items []*itemContext) *tagContext {
	return &tagContext{Tag: tag, Items: items}
}
func (t *tagContext) AbsoluteURL() string {
	return t.Items[0].Section.AbsoluteURL() + "/tag/" + t.Tag + ".html"
}
func (t *tagContext) PageTitle() string {
	return t.Tag + " - " + t.Items[0].Section.Title
}
func (t *tagContext) HomeURL() string {
	return t.Items[0].HomeURL()
}
func (t *tagContext) RootURL() string {
	return t.Items[0].RootURL()
}
func (t *tagContext) HomeTitle() string {
	return t.Items[0].HomeTitle()
}
func (t *tagContext) Include() htpl.HTML {
	return t.Items[0].Include()
}

type itemContext struct {
	Id      int
	Title   string
	Slug    string
	Date    *time.Time
	Tags    []*tagContext
	User    map[string]interface{}
	Section *sectionContext

	item *item
}

func contextFromItem(i *item, s *sectionContext) *itemContext {
	ictx := &itemContext{}
	ictx.Id = i.Id
	ictx.Title = i.Title
	if i.Date == nil {
		ictx.Date = nil
	} else {
		t := i.Date.(time.Time)
		ictx.Date = &t
	}
	var tags []*tagContext
	for _, tag := range i.Tags {
		tags = append(tags, contextFromTag(tag, []*itemContext{ictx}))
	}
	ictx.Tags = tags
	ictx.User = i.User

	ictx.item = i
	ictx.Section = s

	return ictx
}

func (i *itemContext) AbsoluteURL() string {
	return baseDir + strings.TrimPrefix(i.item.outpath, buildDir)
}
func (i *itemContext) PageTitle() string {
	return i.Title + " - " + i.Section.Title
}
func (i *itemContext) HomeURL() string {
	return i.Section.HomeURL()
}
func (i *itemContext) RootURL() string {
	return i.Section.RootURL()
}
func (i *itemContext) HomeTitle() string {
	return i.Section.HomeTitle()
}
func (i *itemContext) Include() htpl.HTML {
	return i.Section.Include()
}
func (i *itemContext) GetBody() htpl.HTML {
	return GetBody(i.item)
}

func outputTemplate(tplname, outpath, style string, cx interface{}) {
	err := os.MkdirAll(filepath.Dir(outpath), 0755)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(outpath)
	if err != nil {
		log.Fatal(err)
	}
	err = getStyleTpl(style).ExecuteTemplate(f, tplname, cx)
	if err != nil {
		log.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func (i *item) needsUpdate() bool {
	if newer(i.inpath, i.outpath) {
		return true
	}
	if newerGlob(stylePath(i.r.s.Style)+"*.html", i.outpath) {
		return true
	}
	for _, d := range i.r.Dependencies {
		tpl := pathToTpl(d, i.r.s.Dir+"/")
		var b bytes.Buffer
		err := tpl.Execute(&b, i)
		if err != nil {
			log.Fatal(err)
		}
		if newerGlob(b.String(), i.outpath) {
			return true
		}
	}
	return false
}

func renderAll() {
	for _, s := range AllSections {
		sctx := contextFromSection(s)
		tags := make(map[string][]*item)
		for _, i := range s.items {
			for _, t := range i.Tags {
				tags[t] = append(tags[t], i)
			}
		}
		var tsctx []*tagContext
		var tagnames []string
		for k, _ := range tags {
			tagnames = append(tagnames, k)
		}
		sort.Strings(tagnames)
		for _, tagname := range tagnames {
			var is []*itemContext
			for _, item := range tags[tagname] {
				is = append(is, contextFromItem(item, sctx))
			}
			tctx := contextFromTag(tagname, is)
			outputTemplate("tag.html", buildDir+s.Dir+"/tag/"+tagname+".html", s.Style, tctx)
			tsctx = append(tsctx, tctx)
		}
		sctx.Tags = tsctx
		outputTemplate("tags.html", buildDir+s.Dir+"/tags.html", s.Style, sctx)

		hasIndex := false
		seenOutpaths := make(map[string]*item)

		for _, i := range s.items {
			if filepath.Base(i.outpath) == "index.html" {
				hasIndex = true
			}

			if seenOutpaths[i.outpath] != nil {
				log.Fatalf("item %q and %q have both outpath %q", seenOutpaths[i.outpath].inpath, i.inpath, i.outpath)
			} else {
				seenOutpaths[i.outpath] = i
			}

			if i.needsUpdate() {
				icx := contextFromItem(i, sctx)
				outputTemplate("single.html", i.outpath, s.Style, icx)
			}
		}

		if !hasIndex {
			SortItemsBy(s.items, "id")
			outputTemplate("index.html", buildDir+s.Dir+"/index.html", s.Style, contextFromSection(s))

		}
	}
}
