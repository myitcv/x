package banana

import "bytes"

//go:generate stateGen

var root _Node_App

type _Node_App struct {
	Model         *bytes.Buffer
	TaggingScreen *_Node_Tagging
}

type _Node_Tagging struct {
	Name string
}
