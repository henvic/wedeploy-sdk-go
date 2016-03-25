package jsonlib

import (
	"fmt"
	"testing"
)

type AssertJSONMarshalProvider struct {
	Want string
	Got  interface{}
	Pass bool
}

type xStruct struct {
	X string `json:"x"`
}

var empty struct{}

var AssertJSONMarshalCases = []AssertJSONMarshalProvider{
	{``, nil, false},
	{`null`, nil, true},
	{`xnull`, nil, false},
	{`"x"`, "x", true},
	{`"x"`, "y", false},
	{`{}`, empty, true},
	{`["y", "z"]`, []string{"y", "z"}, true},
	{`["y", "z"]`, map[string]string{"x": "y"}, false},
	{`null`, map[string]string{"x": "y"}, false},
	{`{"x": "y"}`, xStruct{"y"}, true},
	{`["y", "z"]`, []string{"x", "y", "z"}, false},
}

func TestAssertJSONMarshal(t *testing.T) {
	for _, c := range AssertJSONMarshalCases {
		var mockTest = &testing.T{}
		AssertJSONMarshal(mockTest, c.Want, c.Got)

		if mockTest.Failed() == c.Pass {
			t.Errorf("Mock test did not meet passing status = %v assertion", c.Pass)
			fmt.Println(c.Got)
		}
	}
}