// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"testing"

	protofmt "myitcv.io/protobuf/fmt"

	. "gopkg.in/check.v1"
)

type MainTest struct {
	dir string
}

var _ = Suite(&MainTest{})

func TestMain(t *testing.T) { TestingT(t) }

func (t *MainTest) SetUpTest(c *C) {
	t.dir = tmpDir("protobuf2typescript_test")
}

func (t *MainTest) TearDownTest(c *C) {
	fi, err := os.Stat(t.dir)
	if err == nil && fi.IsDir() {
		os.RemoveAll(t.dir)
	}
}

func (t *MainTest) TestStdoutOutput(c *C) {
	eVar := "PROTOBUF_INCLUDE"
	pbInclude, ok := os.LookupEnv(eVar)
	if !ok {
		log.Fatalf("Could not find %v in ENV", eVar)
	}

	ob := bytes.NewBuffer(nil)

	f := &protofmt.Formatter{
		Output: ob,
	}

	files := []string{"_testFiles/basic.proto"}
	importPaths := []string{"_testFiles/", pbInclude}

	f.Fmt(files, importPaths)

	cmpBytes, err := ioutil.ReadFile("_testFiles/basic.proto.formatted")
	if err != nil {
		panic(err)
	}

	equal := bytes.Equal(ob.Bytes(), cmpBytes)

	if !equal {
		cmd := exec.Command("diff", "-u", "-", "_testFiles/basic.proto.formatted")
		cmd.Stdin = ob
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
		c.Fail()
	} else {
		fmt.Println("===============")
		fmt.Println(string(ob.Bytes()))
		fmt.Println("===============")
	}

}

func tmpDir(prefix string) string {
	outputDir, err := ioutil.TempDir("", prefix)
	if err != nil {
		log.Fatalf("Could not create output dir")
	}

	return outputDir
}
