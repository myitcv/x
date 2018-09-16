package main

import "fmt"

type Elem struct {
	// The myitcv.io/react Name of the element - not set directly, taken from
	// the key of the elements map.
	Name string

	// React is an override for the React name of the element if it is otherwise
	// not equal to the lowercase version of .Name
	React string

	// Dom is the name used by honnef.co/go/js/dom when referring to the underlying
	// HTML element. Default is HTML{{.Name}}Element
	Dom string

	// HTML is an override for the HTML 5 spec name of the element if it is otherwise
	// not equal to the lowercase version of .Name
	HTML string

	// Attributes maps the name of an attribute to the definition of an
	// attribute.
	Attributes map[string]*Attr

	// NonBasic is true if  honnef.co/go/js/dom does not declare a specific
	// Element type.
	NonBasic bool

	// Templates lists the attribute templates this element should use as a
	// base.
	Templates []string

	// NonHTML indicates this element should not automatically inherit the html
	// attribute template
	NonHTML bool

	// Child indicates this element can take a single child of the provided type.
	// Its use is exclusive with Children. No default value.
	Child string

	// Children indicates this element can take a multiple children of the provided
	// type. Its use is exclusive with Child. Default is Element.
	Children string

	// EmptyElement indicates the element may not have any children
	EmptyElement bool

	// Implements is the list of special interface methods this element implements.
	Implements []string

	// SkipTests is an override on whether to not generate the boilerplate tests.
	SkipTests bool
}

func (e *Elem) ChildParam() string {
	if e.Child != "" {
		return "child " + e.Child
	} else if e.Children != "" {
		return "children ..." + e.Children
	}

	return ""
}

func (e *Elem) ChildConvert() string {
	if e.Children != "" && e.Children != "Element" {
		return `
var elems []Element
for _, v := range children {
	elems = append(elems, v)
}
		`
	}

	return ""
}

func (e *Elem) ChildArg() string {
	if e.Child != "" {
		return "child"
	} else if e.Children != "" {
		if e.Children == "Element" {
			return "children..."
		} else {
			return "elems..."
		}
	}

	return ""
}

func (e *Elem) ChildrenReactType() string {
	if e.Children[0] == '*' {
		return "*react." + e.Children[1:]
	}

	return "react." + e.Children
}

func (e *Elem) HTMLAttributes() map[string]*Attr {
	res := make(map[string]*Attr)

	for n, a := range e.Attributes {
		if a.NoHTML || a.NoReact || a.IsEvent || a.Name == "Ref" {
			continue
		}

		res[n] = a
	}

	return res
}

type Attr struct {
	// The myitcv.io/react Name of the attribute - not set directly, taken from
	// the key of the elements map.
	Name string

	// React is an override for the React name of the attribute if it is otherwise
	// not equal to the lower-initial version of .Name
	React string

	// HTML is an override for the HTML attribute name if it is otherwise not equal
	// to the lowercase version of .Name
	HTML string

	// HTMLConvert is a function that must be called on a JSX-parsed value before
	// assignment. Default is nothing.
	HTMLConvert string

	// Type is an override for the type of the attribute. The zero value implies
	// string
	Type string

	// OmitEmpty indicates that no attribute should be set on the underlying React
	// element if the zero value of the attribute is set.
	OmitEmpty bool

	// NoReact indicates that this attribute should not attempt to be mapped directly
	// to an underlying React attribute.
	NoReact bool

	// NoHTML indicates this attribute does not have an HTML equivalent, and hence
	// should not appear during parsing.
	NoHTML bool

	// IsEvent indicates that the attribute is an event.
	IsEvent bool
}

func (a *Attr) Tag() string {
	omitEmpty := ""
	if a.OmitEmpty {
		omitEmpty = ` react:"omitempty"`
	}
	return fmt.Sprintf("`js:\"%v\"%v`", a.React, omitEmpty)
}

func (a *Attr) HTMLConvertor(s string) string {
	if a.HTMLConvert == "" {
		return s
	}

	return fmt.Sprintf("%v(%v)", a.HTMLConvert, s)
}

// templates are the attribute templates to which elements can refer
var templates = map[string]map[string]*Attr{
	"html": {
		"AriaHasPopup":            &Attr{React: "aria-haspopup", Type: "bool", HTML: "aria-haspopup"},
		"AriaExpanded":            &Attr{React: "aria-expanded", Type: "bool", HTML: "aria-expanded"},
		"AriaLabelledBy":          &Attr{React: "aria-labelledby", HTML: "aria-labelledby"},
		"ClassName":               &Attr{HTML: "class"},
		"DangerouslySetInnerHTML": &Attr{Type: "*DangerousInnerHTML", NoHTML: true},
		"DataSet":                 &Attr{Type: "DataSet", NoReact: true},
		"ID":                      &Attr{OmitEmpty: true, React: "id"},
		"Key":                     &Attr{OmitEmpty: true},
		"Ref":                     &Attr{Type: "Ref"},
		"Role":                    &Attr{},
		"Style":                   &Attr{Type: "*CSS", HTMLConvert: "parseCSS"},

		// Events
		"OnBlur":   &Attr{Type: "OnBlur", IsEvent: true},
		"OnFocus":  &Attr{Type: "OnFocus", IsEvent: true},
		"OnChange": &Attr{Type: "OnChange", IsEvent: true},
		"OnClick":  &Attr{Type: "OnClick", IsEvent: true},
	},
}

// elements is a map from the Go element name to the definition
var elements = map[string]*Elem{
	"A": &Elem{
		Dom: "HTMLAnchorElement",
		Attributes: map[string]*Attr{
			"Href":   &Attr{},
			"Target": &Attr{},
			"Title":  &Attr{},
		},
	},
	"Abbr": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Article": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Aside": &Elem{
		Dom: "BasicHTMLElement",
	},
	"B": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Br": &Elem{
		Dom: "HTMLBRElement",
	},
	"Button": &Elem{
		Attributes: map[string]*Attr{
			"AutoFocus":         &Attr{React: "autofocus", Type: "bool", HTML: "autofocus"},
			"Disabled":          &Attr{React: "disabled", Type: "bool", HTML: "disabled"},
			"FormAction":        &Attr{React: "formAction", Type: "string", HTML: "formaction"},
			"FormEncType":       &Attr{React: "formEncType", Type: "string", HTML: "formenctype"},
			"FormMethod":        &Attr{React: "formMethod", Type: "string", HTML: "formmethod"},
			"FormNoValidate":    &Attr{React: "formNoValidate", Type: "bool", HTML: "formnovalidate"},
			"FormTarget":        &Attr{React: "formTarget", Type: "string", HTML: "formtarget"},
			"Name":              &Attr{React: "name", Type: "string", HTML: "name"},
			"TabIndex":          &Attr{React: "tabIndex", Type: "int", HTML: "tabindex"},
			"Type":              &Attr{React: "type", Type: "string", HTML: "type"},
			"ValidationMessage": &Attr{React: "validationMessage", Type: "string", HTML: "validationmessage"},
			"Value":             &Attr{React: "value", Type: "string", HTML: "value"},
			"WillValidate":      &Attr{React: "willValidate", Type: "bool", HTML: "willvalidate"},
		},
	},
	"Caption": &Elem{
		SkipTests: true,
		Dom:       "BasicHTMLElement",
	},
	"Code": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Div": &Elem{},
	"Em": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Footer": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Form": &Elem{
		Attributes: map[string]*Attr{
			"AcceptCharset": &Attr{React: "acceptCharset", Type: "string", HTML: "acceptcharset"},
			"Action":        &Attr{React: "action", Type: "string", HTML: "action"},
			"Autocomplete":  &Attr{React: "autocomplete", Type: "string", HTML: "autocomplete"},
			"Encoding":      &Attr{React: "encoding", Type: "string", HTML: "encoding"},
			"Enctype":       &Attr{React: "enctype", Type: "string", HTML: "enctype"},
			"Length":        &Attr{React: "length", Type: "int", HTML: "length"},
			"Method":        &Attr{React: "method", Type: "string", HTML: "method"},
			"Name":          &Attr{React: "name", Type: "string", HTML: "name"},
			"NoValidate":    &Attr{React: "noValidate", Type: "bool", HTML: "novalidate"},
			"Target":        &Attr{React: "target", Type: "string", HTML: "target"},

			"OnContextMenu": &Attr{Type: "OnContextMenu", IsEvent: true},
			"OnInput":       &Attr{Type: "OnInput", IsEvent: true},
			"OnInvalid":     &Attr{Type: "OnInvalid", IsEvent: true},
			"OnSearch":      &Attr{Type: "OnSearch", IsEvent: true},
			"OnSelect":      &Attr{Type: "OnSelect", IsEvent: true},
			"OnReset":       &Attr{Type: "OnReset", IsEvent: true},
			"OnSubmit":      &Attr{Type: "OnSubmit", IsEvent: true},
		},
	},
	"H1": &Elem{
		Dom: "HTMLHeadingElement",
	},
	"H2": &Elem{
		Dom: "HTMLHeadingElement",
	},
	"H3": &Elem{
		Dom: "HTMLHeadingElement",
	},
	"H4": &Elem{
		Dom: "HTMLHeadingElement",
	},
	"H5": &Elem{
		Dom: "HTMLHeadingElement",
	},
	"H6": &Elem{
		Dom: "HTMLHeadingElement",
	},
	"Header": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Hr": &Elem{
		Dom:          "HTMLHRElement",
		EmptyElement: true,
	},
	"I": &Elem{
		Dom: "BasicHTMLElement",
	},
	"IFrame": &Elem{
		Attributes: map[string]*Attr{
			"Width":    &Attr{React: "width", Type: "string", HTML: "width"},
			"Height":   &Attr{React: "height", Type: "string", HTML: "height"},
			"Name":     &Attr{React: "name", Type: "string", HTML: "name"},
			"Src":      &Attr{React: "src", Type: "string", HTML: "src"},
			"SrcDoc":   &Attr{React: "srcdoc", Type: "string", HTML: "srcdoc"},
			"Seamless": &Attr{React: "seamless", Type: "bool", HTML: "seamless"},
		},
	},
	"Img": &Elem{
		Dom: "HTMLImageElement",
		Attributes: map[string]*Attr{
			"Alt":           &Attr{},
			"Complete":      &Attr{React: "complete", Type: "bool", HTML: "complete"},
			"CrossOrigin":   &Attr{React: "crossOrigin", Type: "string", HTML: "crossorigin"},
			"Height":        &Attr{React: "height", Type: "int", HTML: "height"},
			"IsMap":         &Attr{React: "isMap", Type: "bool", HTML: "ismap"},
			"NaturalHeight": &Attr{React: "naturalHeight", Type: "int", HTML: "naturalheight"},
			"NaturalWidth":  &Attr{React: "naturalWidth", Type: "int", HTML: "naturalwidth"},
			"Src":           &Attr{React: "src", Type: "string", HTML: "src"},
			"UseMap":        &Attr{React: "useMap", Type: "string", HTML: "usemap"},
			"Width":         &Attr{React: "width", Type: "int", HTML: "width"},
		},
	},
	"Input": &Elem{
		Attributes: map[string]*Attr{
			"Accept":             &Attr{React: "accept", Type: "string", HTML: "accept"},
			"Alt":                &Attr{React: "alt", Type: "string", HTML: "alt"},
			"Autocomplete":       &Attr{React: "autocomplete", Type: "string", HTML: "autocomplete"},
			"Autofocus":          &Attr{React: "autofocus", Type: "bool", HTML: "autofocus"},
			"Checked":            &Attr{React: "checked", Type: "bool", HTML: "checked"},
			"DefaultChecked":     &Attr{React: "defaultChecked", Type: "bool", HTML: "defaultchecked"},
			"DefaultValue":       &Attr{React: "defaultValue", Type: "string", HTML: "defaultvalue"},
			"DirName":            &Attr{React: "dirName", Type: "string", HTML: "dirname"},
			"Disabled":           &Attr{React: "disabled", Type: "bool", HTML: "disabled"},
			"FormAction":         &Attr{React: "formAction", Type: "string", HTML: "formaction"},
			"FormEncType":        &Attr{React: "formEncType", Type: "string", HTML: "formenctype"},
			"FormMethod":         &Attr{React: "formMethod", Type: "string", HTML: "formmethod"},
			"FormNoValidate":     &Attr{React: "formNoValidate", Type: "bool", HTML: "formnovalidate"},
			"FormTarget":         &Attr{React: "formTarget", Type: "string", HTML: "formtarget"},
			"Height":             &Attr{React: "height", Type: "string", HTML: "height"},
			"Indeterminate":      &Attr{React: "indeterminate", Type: "bool", HTML: "indeterminate"},
			"Max":                &Attr{React: "max", Type: "string", HTML: "max"},
			"MaxLength":          &Attr{React: "maxLength", Type: "int", HTML: "maxlength"},
			"Min":                &Attr{React: "min", Type: "string", HTML: "min"},
			"Multiple":           &Attr{React: "multiple", Type: "bool", HTML: "multiple"},
			"Name":               &Attr{React: "name", Type: "string", HTML: "name"},
			"Pattern":            &Attr{React: "pattern", Type: "string", HTML: "pattern"},
			"Placeholder":        &Attr{React: "placeholder", Type: "string", HTML: "placeholder"},
			"ReadOnly":           &Attr{React: "readOnly", Type: "bool", HTML: "readonly"},
			"Required":           &Attr{React: "required", Type: "bool", HTML: "required"},
			"SelectionDirection": &Attr{React: "selectionDirection", Type: "string", HTML: "selectiondirection"},
			"SelectionEnd":       &Attr{React: "selectionEnd", Type: "int", HTML: "selectionend"},
			"SelectionStart":     &Attr{React: "selectionStart", Type: "int", HTML: "selectionstart"},
			"Size":               &Attr{React: "size", Type: "int", HTML: "size"},
			"Src":                &Attr{React: "src", Type: "string", HTML: "src"},
			"Step":               &Attr{React: "step", Type: "string", HTML: "step"},
			"TabIndex":           &Attr{React: "tabIndex", Type: "int", HTML: "tabindex"},
			"Type":               &Attr{React: "type", Type: "string", HTML: "type"},
			"ValidationMessage":  &Attr{React: "validationMessage", Type: "string", HTML: "validationmessage"},
			"Value":              &Attr{React: "value", Type: "string", HTML: "value"},
			"ValueAsDate":        &Attr{React: "valueAsDate", Type: "time", HTML: "valueasdate"},
			"ValueAsNumber":      &Attr{React: "valueAsNumber", Type: "float64", HTML: "valueasnumber"},
			"Width":              &Attr{React: "width", Type: "string", HTML: "width"},
			"WillValidate":       &Attr{React: "willValidate", Type: "bool", HTML: "willvalidate"},
		},
	},
	"Label": &Elem{
		Attributes: map[string]*Attr{
			"For": &Attr{
				React: "htmlFor",
			},
		},
	},
	"Li": &Elem{
		Dom:        "HTMLLIElement",
		Implements: []string{"RendersLi(*LiElem)"},
	},
	"Main": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Nav": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Option": &Elem{
		Attributes: map[string]*Attr{
			"DefaultSelected": &Attr{React: "defaultSelected", Type: "bool", HTML: "defaultselected"},
			"Disabled":        &Attr{React: "disabled", Type: "bool", HTML: "disabled"},
			"Index":           &Attr{React: "index", Type: "int", HTML: "index"},
			"Label":           &Attr{React: "label", Type: "string", HTML: "label"},
			"Selected":        &Attr{React: "selected", Type: "bool", HTML: "selected"},
			"Text":            &Attr{React: "text", Type: "string", HTML: "text"},
			"Value":           &Attr{React: "value", Type: "string", HTML: "value"},
		},
	},
	"P": &Elem{
		Dom: "HTMLParagraphElement",
	},
	"Pre": &Elem{},
	"Select": &Elem{
		Attributes: map[string]*Attr{
			"Value": &Attr{},
		},
		Children: "*OptionElem",
	},
	"Span": &Elem{},
	"Strike": &Elem{
		Dom:   "BasicHTMLElement",
		React: "s",
		HTML:  "s",
	},
	"Sup": &Elem{
		Dom: "BasicHTMLElement",
	},
	"Table": &Elem{},
	"Tbody": &Elem{
		SkipTests: true,
		Dom:       "BasicHTMLElement",
	},
	"Td": &Elem{
		SkipTests: true,
		Dom:       "BasicHTMLElement",
	},
	"TextArea": &Elem{
		Attributes: map[string]*Attr{
			"Autocomplete":       &Attr{React: "autocomplete", Type: "string", HTML: "autocomplete"},
			"Autofocus":          &Attr{React: "autofocus", Type: "bool", HTML: "autofocus"},
			"Cols":               &Attr{React: "cols", Type: "int", HTML: "cols"},
			"DefaultValue":       &Attr{React: "defaultValue", Type: "string", HTML: "defaultvalue"},
			"DirName":            &Attr{React: "dirName", Type: "string", HTML: "dirname"},
			"Disabled":           &Attr{React: "disabled", Type: "bool", HTML: "disabled"},
			"MaxLength":          &Attr{React: "maxLength", Type: "int", HTML: "maxlength"},
			"Name":               &Attr{React: "name", Type: "string", HTML: "name"},
			"Placeholder":        &Attr{React: "placeholder", Type: "string", HTML: "placeholder"},
			"ReadOnly":           &Attr{React: "readOnly", Type: "bool", HTML: "readonly"},
			"Required":           &Attr{React: "required", Type: "bool", HTML: "required"},
			"Rows":               &Attr{React: "rows", Type: "int", HTML: "rows"},
			"SelectionDirection": &Attr{React: "selectionDirection", Type: "string", HTML: "selectiondirection"},
			"SelectionStart":     &Attr{React: "selectionStart", Type: "int", HTML: "selectionstart"},
			"SelectionEnd":       &Attr{React: "selectionEnd", Type: "int", HTML: "selectionend"},
			"TabIndex":           &Attr{React: "tabIndex", Type: "int", HTML: "tabindex"},
			"TextLength":         &Attr{React: "textLength", Type: "int", HTML: "textlength"},
			"Type":               &Attr{React: "type", Type: "string", HTML: "type"},
			"ValidationMessage":  &Attr{React: "validationMessage", Type: "string", HTML: "validationmessage"},
			"Value":              &Attr{React: "value", Type: "string", HTML: "value"},
			"WillValidate":       &Attr{React: "willValidate", Type: "bool", HTML: "willvalidate"},
			"Wrap":               &Attr{React: "wrap", Type: "string", HTML: "wrap"},
		},
	},
	"Th": &Elem{
		SkipTests: true,
		Dom:       "BasicHTMLElement",
	},
	"Thead": &Elem{
		SkipTests: true,
		Dom:       "BasicHTMLElement",
	},
	"Tr": &Elem{
		SkipTests: true,
		Dom:       "BasicHTMLElement",
	},
	"Ul": &Elem{
		Dom:      "HTMLUListElement",
		Children: "RendersLi",
	},
}
