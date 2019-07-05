package main

//go:generate gobin -m -run myitcv.io/react/cmd/reactGen

import (
	"path/filepath"

	r "myitcv.io/react"
)

type PreviewProps struct {
	Current *Gopher
}

type PreviewDef struct {
	r.ComponentDef
}

func Preview(p PreviewProps) *PreviewElem {
	return buildPreviewElem(p)
}

func (o PreviewDef) Render() r.Element {
	var parts []r.Element

	curr := o.Props().Current

	addPart := func(p string) {
		parts = append(parts, r.Img(&r.ImgProps{
			Src: filepath.Join("artwork", p+".png"),
			Style: &r.CSS{
				MarginTop: "0px",
			},
		}))
	}

	for _, p := range curr.Parts {
		if p != "" {
			addPart(p)
		}
	}

	return r.Div(&r.DivProps{ClassName: "col-xs-8"},
		r.Div(&r.DivProps{ID: "preview"},
			parts...,
		),
	)
}
