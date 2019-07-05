package main

//go:generate gobin -m -run myitcv.io/react/cmd/reactGen

import (
	"github.com/gopherjs/gopherjs/js"
	r "myitcv.io/react"
	"myitcv.io/react/jsx"
)

type ChooserProps struct {
	Current *Gopher
	Config  *Config
	Update  UpdateGopher
}

type ChooserState struct {
	open int
}

type ChooserDef struct {
	r.ComponentDef
}

func Chooser(p ChooserProps) *ChooserElem {
	return buildChooserElem(p)
}

func (ch ChooserDef) Render() r.Element {
	var catDivs []r.Element

	st := ch.State()
	props := ch.Props()
	cg := props.Current

	for i, cat := range ch.Props().Config.Categories {
		catDivs = append(catDivs, Panel(
			PanelProps{
				Category: cat,
				Open:     st.open == i,
				Part:     i,
				Selected: cg.Parts[i],
				Update:   props.Update,
				Expand:   ch,
			},
		))
	}

	args := []r.Element{
		r.Button(
			&r.ButtonProps{
				ID:        "shuffle-button",
				ClassName: "btn btn-default",
				OnClick:   shuffleClick{ch},
			},
			r.I(&r.IProps{ClassName: "glyphicon glyphicon-refresh"}),
			r.S(" Shuffle"),
		),
		r.Button(
			&r.ButtonProps{
				ID:        "reset-button",
				ClassName: "btn btn-default",
				OnClick:   resetClick{ch},
			},
			r.S("Reset"),
		),
		r.Br(nil),
		r.Br(nil),
		r.Div(
			&r.DivProps{
				ClassName: "panel-group",
				ID:        "options",
				Role:      "tablist",
			},
			catDivs...,
		),
		r.Div(&r.DivProps{ClassName: "panel panel-default"},
			r.Div(
				&r.DivProps{
					ClassName: "panel-body text-right",
					Style:     &r.CSS{OverflowY: "hidden"},
				},
				r.Button(
					&r.ButtonProps{
						ID:        "next-button",
						ClassName: "btn btn-primary btn-lg",
						OnClick:   saveClick{ch},
					},
					r.S("Save \u0026 continue\u2026"),
					r.I(&r.IProps{ClassName: "glyphicon glyphicon glyphicon-chevron-right"}),
				),
			),
		),
		jsx.HTMLElem(`
			<footer>
				Be truly unique, there are
				<span class='total_combinations'></span>
				<hr/>
				Artwork by <a href='https://twitter.com/ashleymcnamara' target='_blank'>Ashley McNamara</a><br />inspired by <a href='http://reneefrench.blogspot.co.uk/' target='_blank'>Renee French</a><br />
				Original web app by <a href='https://twitter.com/matryer' target='_blank'>Mat Ryer</a><br/>
				Front-end Go React version by <a href="https://twitter.com/_myitcv" target="_blank">Paul Jolly</a>
				<hr>
				<a href='https://github.com/myitcv/gopherize.me/tree/master/client'>Source on GitHub</a><br/>
				<a href='https://github.com/matryer/gopherize.me'>Original source on GitHub</a>
			</footer>
		`),
	}

	return r.Div(&r.DivProps{ClassName: "col-xs-4"}, args...)
}

func (ch ChooserDef) Expand(i int) {
	s := ch.State()
	s.open = i
	ch.SetState(s)
}

type ExpandPanel interface {
	Expand(i int)
}

type shuffleClick struct{ ChooserDef }

func (sh shuffleClick) OnClick(e *r.SyntheticMouseEvent) {
	sh.Props().Update.RandomGopher()
	e.PreventDefault()
}

type resetClick struct{ ChooserDef }

func (rc resetClick) OnClick(e *r.SyntheticMouseEvent) {
	rc.Props().Update.ResetGopher()
	e.PreventDefault()
}

type saveClick struct{ ChooserDef }

func (sc saveClick) OnClick(e *r.SyntheticMouseEvent) {
	js.Global.Call("alert", "Frontend only for now...")
	e.PreventDefault()
}
