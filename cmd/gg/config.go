package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

const (
	ConfigFileName = ".ggconfig.json"
)

type Config struct {
	Cmds    []string
	NonCmds []string

	cmds     map[string]struct{}
	baseCmds map[string]string
	nonCmds  map[string]struct{}
}

var config Config

func loadConfig() {
	// TODO maybe instead of using $PWD as the starting point for finding a config file we should start at
	// the package directory...

	var err error

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dir := wd

	var fi *os.File

	for {
		f := filepath.Join(dir, ConfigFileName)

		fi, err = os.Open(f)
		if err == nil {
			break
		}

		p := filepath.Dir(dir)

		if p == dir {
			break
		}

		dir = p
	}

	if fi == nil {
		log.Fatalf("Could not find %v in %v (or any parent directory)", ConfigFileName, wd)
	}

	j := json.NewDecoder(fi)
	err = j.Decode(&config)
	if err != nil {
		log.Fatalf("Could not decode config file %v:\n%v", fi.Name(), err)
	}

	config.cmds = make(map[string]struct{})
	config.baseCmds = make(map[string]string)
	config.nonCmds = make(map[string]struct{})

	for _, v := range config.Cmds {
		b := filepath.Base(v)
		config.cmds[v] = struct{}{}
		config.baseCmds[b] = v
	}

	for _, v := range config.NonCmds {
		config.nonCmds[v] = struct{}{}
	}

	config.Cmds = nil
	for k := range config.cmds {
		config.Cmds = append(config.Cmds, k)
	}

	config.NonCmds = nil
	for k := range config.nonCmds {
		config.NonCmds = append(config.NonCmds, k)
	}
}
