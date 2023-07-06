package lib

import "encoding/json"

type Context struct {
	Template      Template                   `json:"template"`
	Configuration Configuration              `json:"configuration"`
	Values        []Values                   `json:"values,omitempty"`
	Schemas       map[string]json.RawMessage `json:"schemas,omitempty"`
}
type Template struct {
	Location string `json:"location"`
	Header   string `json:"header"`
	Footer   string `json:"footer"`
}

type Values struct {
	Data []byte `json:"data"`
	Type string `json:"type"`
}

type Configuration struct {
	StrictUndefined bool       `json:"strict_undefined"`
	Delimiters      Delimiters `json:"delimiters"`
}
type Delimiters struct {
	BlockStart    string `json:"block_start"`
	BlockEnd      string `json:"block_end"`
	VariableStart string `json:"variable_start"`
	VariableEnd   string `json:"variable_end"`
	CommentStart  string `json:"comment_start"`
	CommentEnd    string `json:"comment_end"`
}
