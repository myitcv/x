package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"myitcv.io/cmd/mdreplace/internal/itemtype"
)

func (p *processor) processCommonBlock(prefix string, conv func([]byte) interface{}) procFn {
	// consume the (quoted) arguments

	var orig []string
	var args []string
	var options []string

	execute := true

Args:
	for {
		i := p.next()

		t := i.val

		switch i.typ {
		case itemtype.ItemArgComment:
			options = []string{}
			// consume any options
			for {
				i := p.next()
				if i.typ != itemtype.ItemOption {
					break Args
				}
				switch i.val {
				case optionLong:
					execute = execute && *fLong
				case optionOnline:
					execute = execute && *fOnline
				default:
					p.errorf("unknown option %v", i.val)
				}
				options = append(options, i.val)
			}

		case itemtype.ItemArg:
		case itemtype.ItemQuoteArg:
			v, err := strconv.Unquote(i.val)
			if err != nil {
				p.errorf("failed to unquote %q: %v", i.val, err)
			}
			t = v
		default:
			break Args
		}

		orig = append(orig, i.val)

		// this should succeed because we previously unquoted it during lexing

		t = os.Expand(t, func(s string) string {
			debugf("Expand %q\n", s)
			if s == "DOLLAR" {
				return "$"
			}

			return os.Getenv(s)
		})

		args = append(args, t)
	}

	debugf("Will run with args \"%v\"\n", strings.Join(args, "\", \""))

	origCmdStr := strings.Join(orig, " ")

	if len(args) == 0 {
		p.errorf("didn't see any args")
	}

	// at this point we can accept a run of text or code fence blocks
	// because both are valid as block args; we simple concat them
	// together
	tmpl := new(strings.Builder)

	for p.curr.typ != itemtype.ItemCommEnd {
		switch p.curr.typ {
		case itemtype.ItemCodeFence, itemtype.ItemCode, itemtype.ItemText:
			tmpl.WriteString(p.curr.val)
		default:
			p.errorf("didn't expect to see a %v", p.curr.typ)
		}
		p.next()
	}

	// consume the commEnd
	p.next()

	// print the header now in case we are not executing
	if !*fStrip {
		p.printf(prefix+" %v", origCmdStr)

		if len(options) > 0 {
			p.printf(" %v %v", string(optionStart), strings.Join(options, " "))
		}

		p.printf("\n%v"+commEnd+"\n", tmpl)
	}

	// again we can expect text or code fence blocks here; we are just
	// going to ignore them.
	for p.curr.typ != itemtype.ItemBlockEnd {
		switch p.curr.typ {
		case itemtype.ItemCodeFence, itemtype.ItemCode, itemtype.ItemText:
			if !execute {
				p.print(p.curr.val)
			}
		default:
			p.errorf("didn't expect to see a %v", p.curr.typ)
		}
		p.next()
	}

	// consume the block end
	p.next()

	if execute {
		// ok now process the command, parse the template and write everything
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			p.errorf("failed to run command %q: %v\n%v", origCmdStr, err, string(out))
		}

		t, err := template.New("").Funcs(tmplFuncMap).Parse(tmpl.String())
		if err != nil {
			p.errorf("failed to parse template %q: %e", tmpl, err)
		}

		i := conv(out)

		if err := t.Execute(p.out, i); err != nil {
			p.errorf("failed to execute template %q with input %q: %v", tmpl, i, err)
		}
	}

	if !*fStrip {
		p.println(blockEnd)
	}

	return p.processText
}
