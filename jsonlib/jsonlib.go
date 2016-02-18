package jsonlib

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

// AssertJSONMarshal asserts that a given object marshals to JSON correctly
func AssertJSONMarshal(t *testing.T, want string, got interface{}) {
	var wantJSON interface{}
	var gotMap interface{}

	bin, err := json.Marshal(got)

	if err != nil {
		t.Error(err)
	}

	if err = json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Errorf("Wanted value %s isn't JSON.", want)
	}

	if err = json.Unmarshal(bin, &gotMap); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(wantJSON, gotMap) {
		t.Errorf("JSON doesn't have the same structure:\n%s",
			pretty.Compare(wantJSON, gotMap))
	}
}
