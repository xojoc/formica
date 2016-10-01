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
	htpl "html/template"
	"log"
	"path/filepath"
	"strings"
	"time"
)

const stylesDir string = "styles"

var styleDef string = "default"
var styleTpls = make(map[string]*htpl.Template)

func stylePath(style string) string {
	return cfgDir + "/" + stylesDir + "/" + style + "/"
}

func newStyleTpl(style string) *htpl.Template {
	mapFunc := htpl.FuncMap{
		"GetBody":     GetBody,
		"Exec":        Exec,
		"DateFormat":  DateFormat,
		"SortItemsBy": SortItemsBy,
		"RawHtml":     RawHtml,
	}
	styletpl, err := htpl.New(style).Funcs(mapFunc).ParseGlob(stylePath(style) + "/" + "*.html")
	if err != nil {
		log.Fatal(err)
	}
	return styletpl
}

func getStyleTpl(style string) *htpl.Template {
	if s, ok := styleTpls[style]; ok {
		return s
	}
	styleTpls[style] = newStyleTpl(style)
	return styleTpls[style]
}

func Exec(cmdstr string) htpl.HTML {
	var buf bytes.Buffer
	execShellIO(cmdstr, nil, &buf, nil)
	return htpl.HTML(buf.String())
}

func RawHtml(s string) htpl.HTML {
	return htpl.HTML(s)
}

func DateFormat(d *time.Time, layout string) string {
	if layout == "" {
		layout = "2006-01-02"
	}
	return (*d).Format(layout)
}

// FIXME: copy assets, favicon, etc.
func copyCss() {
	for _, s := range AllSections {
		fs, err := filepath.Glob(stylePath(s.Style) + "*.css")
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range fs {
			copyFile(f, buildDir+"/css/"+strings.TrimPrefix(f, cfgDir+"/"+stylesDir+"/"))
		}
	}
}

func copyJs() {
	for _, s := range AllSections {
		fs, err := filepath.Glob(stylePath(s.Style) + "*.js")
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range fs {
			copyFile(f, buildDir+"/js/"+strings.TrimPrefix(f, cfgDir+"/"+stylesDir+"/"))
		}
	}
}

func copyAssets() {
	copyCss()
	copyJs()
}
