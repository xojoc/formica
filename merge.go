package main

import (
	"io"
)

type MergeInput struct {
	src []io.Reader
}

func (mi *MergeInput) Read(p []byte) (n int, err error) {
	for len(mi.src) > 0 {
		n, err = mi.src[0].Read(p)
		if err != nil {
			if err == io.EOF {
				err = nil
			} else {
				return
			}
		}

		if n > 0 {
			return
		} else {
			mi.src = mi.src[1:]
		}
	}

	return 0, io.EOF
}

func (mi *MergeInput) AddSource(r io.Reader) {
	mi.src = append(mi.src, r)
}

func NewMergeInput(rs ...io.Reader) *MergeInput {
	mi := &MergeInput{}
	for _, r := range rs {
		mi.AddSource(r)
	}
	return mi
}
