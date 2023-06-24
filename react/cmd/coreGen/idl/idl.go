package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type ProcessedIDL struct {
	Interfaces *InterfacesIDL `json:"interfaces"`
}

type InterfacesIDL struct {
	Interface map[string]*InterfaceIDL `json:"interface"`
}

type InterfaceIDL struct {
	Name       string         `json:"name"`
	Elements   []ElementIDL   `json:"element"`
	Properties *PropertiesIDL `json:"properties"`
	Extends    string         `json:"extends"`
}

type PropertiesIDL struct {
	Property map[string]*PropertyIDL `json:"property"`
}

type PropertyIDL struct {
	Name string
	Type PropertyType
}

func (p *PropertyIDL) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name     string          `json:"name"`
		Elements []ElementIDL    `json:"element"`
		Type     json.RawMessage `json:"type"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.Name = raw.Name

	{
		// Type could be a string ,,,
		var tn string
		if err := json.Unmarshal(raw.Type, &tn); err == nil {
			p.Type = PropertyTypeInst{
				Type: tn,
			}
			goto DoneType
		}

		// ...or []PropertyTypeInst
		var typs PropertyTypeInsts
		if err := json.Unmarshal(raw.Type, &typs); err == nil {
			p.Type = typs
			goto DoneType
		}

		return fmt.Errorf("failed to parse property type from [%s]", raw.Type)
	}

DoneType:

	return nil
}

type ElementIDL struct {
	Specs     string `json:"specs"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type PropertyType interface {
	isPropertyType()
}

type PropertyTypeInst struct {
	Type     string
	Nullable bool
}

func (p PropertyTypeInst) isPropertyType() {}

type PropertyTypeInsts []PropertyTypeInst

func (p PropertyTypeInsts) isPropertyType() {}

func (i *PropertyTypeInst) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type     string `json:"type"`
		Nullable int    `json:"nullable"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	i.Type = raw.Type
	i.Nullable = raw.Nullable == 1 // appears to be the convention

	return nil
}

// ---

func main() {
	os.Exit(main1())
}

func main1() int {
	if err := mainerr(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func mainerr() error {
	printOut := make(map[string]bool)
	for _, v := range []string{
		"HTMLTableElement",
		"HTMLDivElement",
		"HTMLElement",
		"Element",
		"Node",
	} {
		printOut[v] = true
	}
	flag.Parse()

	fn := flag.Arg(0)

	fi, err := os.Open(fn)
	if err != nil {
		return fmt.Errorf("failed to open %v: %v", fn, err)
	}

	dec := json.NewDecoder(fi)
	var p ProcessedIDL

	if err := dec.Decode(&p); err != nil {
		return fmt.Errorf("failed to parse %v: %v", fn, err)
	}

	for k, v := range p.Interfaces.Interface {
		if !printOut[k] {
			continue
		}

		fmt.Println("==========")

		fmt.Printf("%v: %v\n", k, v.Name)

		fmt.Printf("extends> %v\n", v.Extends)

		for _, v := range v.Elements {
			fmt.Printf("element> %v\n", v.Name)
		}

		for k, v := range v.Properties.Property {
			fmt.Printf("prop> %v: %#v\n", k, v)
		}
	}

	return nil
}
