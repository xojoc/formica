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
	"fmt"
	htpl "html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"xojoc.pw/must"
)

const (
	baseDir  = "/"
	rssPath  = "/rss"
	atomPath = "/atom"
)

type sectionContext struct {
	Dir         string
	Title       string
	Excerpt     string
	Tags        []*tagContext
	Items       []*itemContext
	TagsContext bool

	section *section
}

func contextFromSection(s *section) *sectionContext {
	sctx := &sectionContext{}
	sctx.Dir = s.Dir
	sctx.Title = s.Title
	sctx.Excerpt = s.Excerpt
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
	p := filepath.Clean(baseDir + s.Dir)
	if p == "/" {
		return p
	}
	if s.TagsContext {
		return p + "/tags"
	}
	return p + "/"
}
func (s *sectionContext) FeedURL() string {
	if s.section.Feed {
		return baseDir + s.Dir + atomPath
	}
	return ""
}
func (s *sectionContext) HomeURL() string {
	p := filepath.Clean(s.AbsoluteURL())
	if s.TagsContext {
		p = strings.TrimSuffix(p, "tags")
	}
	return p
}
func (s *sectionContext) RootURL() string {
	return baseDir
}
func (s *sectionContext) HomeTitle() string {
	return s.Title
}
func (s *sectionContext) GoPath() string {
	return ""
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

	if s.section.Feed {
		str += fmt.Sprintf(`<link rel="alternate" type="application/rss+xml" href="%s%s" />`, s.AbsoluteURL(), rssPath)
		str += fmt.Sprintf(`<link rel="alternate" type="application/atom+xml" href="%s%s" />`, s.AbsoluteURL(), atomPath)
	}

	return htpl.HTML(str)
}

type tagContext struct {
	Tag     string
	Items   []*itemContext
	Excerpt string
}

func contextFromTag(tag string, items []*itemContext) *tagContext {
	return &tagContext{Tag: tag, Items: items}
}
func (t *tagContext) AbsoluteURL() string {
	p := t.Items[0].Section.AbsoluteURL()
	if strings.HasSuffix(p, "/tags") {
		p = p + "/.."
	}
	return filepath.Clean(p + "/tag/" + t.Tag)
}
func (t *tagContext) FeedURL() string {
	return t.Items[0].FeedURL()
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
func (t *tagContext) GoPath() string {
	return ""
}

type itemContext struct {
	Id      int
	Title   string
	Excerpt string
	Slug    string
	Date    *time.Time
	Tags    []*tagContext
	User    map[string]interface{}
	Section *sectionContext

	GoPath          string
	GoCode          string
	GoDocumentation string

	item *item
}

func contextFromItem(i *item, s *sectionContext) *itemContext {
	ictx := &itemContext{}
	ictx.Id = i.Id
	ictx.Title = i.Title
	ictx.Excerpt = i.Excerpt
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
	sort.Slice(tags, func(i, j int) bool { return strings.ToLower(tags[i].Tag) < strings.ToLower(tags[j].Tag) })
	ictx.Tags = tags
	ictx.User = i.User

	ictx.item = i
	ictx.Section = s

	ictx.GoPath = i.GoPath
	ictx.GoCode = i.GoCode

	return ictx
}

func (i *itemContext) AbsoluteURL() string {
	return baseDir + strings.TrimSuffix(strings.TrimPrefix(i.item.outpath, buildDir), ".html")
}
func (i *itemContext) FeedURL() string {
	return i.Section.FeedURL()
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

func outputFeeds(s *sectionContext) {
	now := time.Now()
	feed := &feeds.Feed{
		Title:   s.HomeTitle() + " - Xojoc",
		Link:    &feeds.Link{Href: s.AbsoluteURL()},
		Created: now,
	}
	for _, i := range s.Items {
		var date time.Time
		if i.Date != nil {
			date = *i.Date
		}
		feed.Items = append(feed.Items, &feeds.Item{
			Title:   i.Title,
			Link:    &feeds.Link{Href: i.AbsoluteURL()},
			Created: date,
		})
	}

	sort.Slice(feed.Items, func(i, j int) bool {
		return feed.Items[i].Created.After(feed.Items[j].Created)
	})

	atom, err := feed.ToAtom()
	must.OK(err)

	rss, err := feed.ToRss()
	must.OK(err)

	must.OK(ioutil.WriteFile(buildDir+s.Dir+atomPath, []byte(atom), 0755))
	must.OK(ioutil.WriteFile(buildDir+s.Dir+rssPath, []byte(rss), 0755))
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

func outputSitemap(sitemap []string) {
	f := must.Create(buildDir + "/sitemap.xml")
	_, err := f.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">` + "\n")
	must.OK(err)

	for _, l := range sitemap {
		_, err := f.WriteString(`<url><loc>` + Config.SiteURL + l + `</loc></url>` + "\n")
		must.OK(err)
	}

	_, err = f.WriteString(`</urlset>` + "\n")
	must.OK(err)
	must.Close(f)
}

func renderAll() {
	var sitemap []string

	for _, s := range AllSections {
		sctx := contextFromSection(s)
		sitemap = append(sitemap, sctx.AbsoluteURL())
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
		sort.Slice(tagnames, func(i, j int) bool { return strings.ToLower(tagnames[i]) < strings.ToLower(tagnames[j]) })
		for _, tagname := range tagnames {
			var is []*itemContext
			items :=  tags[tagname]
			SortItemsBy(items, s.IndexSort)
			for _, item := range items {
				is = append(is, contextFromItem(item, sctx))
			}
			tctx := contextFromTag(tagname, is)
			outputTemplate("tag.html", buildDir+s.Dir+"/tag/"+tagname+".html", s.Style, tctx)
			tsctx = append(tsctx, tctx)

			sitemap = append(sitemap, tctx.AbsoluteURL())
		}
		sctx.Tags = tsctx
		sctx.TagsContext = true
		outputTemplate("tags.html", buildDir+s.Dir+"/tags.html", s.Style, sctx)
		sctx.TagsContext = false

		hasIndex := false
		seenOutpaths := make(map[string]*item)

		for _, i := range s.items {
			if i.GoPath != "" {
				f := must.Open(os.Getenv("GOPATH") + "/src/" + i.GoPath + "/README.md")
				i.buf = bufio.NewReader(io.MultiReader(f, i.buf))
				if i.Title == "" {
					i.Title = i.GoPath
				}
			}

			if filepath.Base(i.outpath) == "index.html" {
				hasIndex = true
			}

			if seenOutpaths[i.outpath] != nil {
				log.Fatalf("item %q and %q have both outpath %q", seenOutpaths[i.outpath].inpath, i.inpath, i.outpath)
			} else {
				seenOutpaths[i.outpath] = i
			}

			icx := contextFromItem(i, sctx)
			if i.needsUpdate() {
				outputTemplate("single.html", i.outpath, s.Style, icx)
			}

			sitemap = append(sitemap, icx.AbsoluteURL())
		}

		if !hasIndex {
			SortItemsBy(s.items, s.IndexSort)
			outputTemplate("index.html", buildDir+s.Dir+"/index.html", s.Style, contextFromSection(s))
		}

		if s.Feed {
			outputFeeds(sctx)
		}
	}

	outputSitemap(sitemap)
}
