package lib

import (
	"encoding/json"
	"time"
)

type Context struct {
	Source        Source                     `json:"template"`
	Configuration Configuration              `json:"configuration"`
	Values        []Values                   `json:"values,omitempty"`
	Schemas       map[string]json.RawMessage `json:"schemas,omitempty"`
	Timeout       time.Duration              `json:"render_timeout,omitempty"`
}
type Source struct {
	Template  string `json:"content"`
	Directory string `json:"directory"`
}

type Values struct {
	Data []byte `json:"data"`
	Type string `json:"type"`
}

type Configuration struct {
	StrictUndefined bool       `json:"strict_undefined"`
	Delimiters      Delimiters `json:"delimiters"`
	LeftStripBlocks bool       `json:"left_strip_blocks"`
	TrimBlocks      bool       `json:"trim_blocks"`
}
type Delimiters struct {
	BlockStart    string `json:"block_start"`
	BlockEnd      string `json:"block_end"`
	VariableStart string `json:"variable_start"`
	VariableEnd   string `json:"variable_end"`
	CommentStart  string `json:"comment_start"`
	CommentEnd    string `json:"comment_end"`
}
