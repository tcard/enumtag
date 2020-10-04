package enumtag_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tcard/enumtag"
)

func TestUnmarshalMarshal(t *testing.T) {
	type enumType struct {
		Value    interface{} `enumvaluefield:"value"`
		Variants [0]*struct {
			Foo string
			Bar int
			Qux []string `enumtag:"qux"`
		} `enumtagfield:"type"`
	}

	type ABC struct {
		A, B, C string
	}

	type DEF struct {
		D, E, F int
	}

	type enumTypeEmbeddedValue struct {
		Value    interface{} `enumvaluefield:"-"`
		Variants [0]*struct {
			ABC `enumtag:"abc"`
			DEF `enumtag:"def"`
		} `enumtagfield:"type"`
	}

	for _, tc := range []struct {
		name          string
		enum          interface{}
		json          string
		expected      interface{}
		expectedError string
	}{{
		name:     "ok string",
		enum:     enumType{},
		json:     `{"type": "Foo", "value": "foo"}`,
		expected: "foo",
	}, {
		name:     "ok int",
		enum:     enumType{},
		json:     `{"type": "Bar", "value": 123}`,
		expected: 123,
	}, {
		name:     "ok explicit tag",
		enum:     enumType{},
		json:     `{"type": "qux", "value": ["a", "b", "c"]}`,
		expected: []string{"a", "b", "c"},
	}, {
		name:          "unknown tag",
		enum:          enumType{},
		json:          `{"type": "unknown", "value": ["a", "b", "c"]}`,
		expectedError: `unknown tag "unknown"`,
	}, {
		name:          "bad value",
		enum:          enumType{},
		json:          `{"type": "qux", "value": "not a slice"}`,
		expectedError: `unmarshaling enum value into type []string`,
	}, {
		name:     "ok embedded ABC",
		enum:     enumTypeEmbeddedValue{},
		json:     `{"type": "abc", "A": "foo", "B": "bar", "C": "qux"}`,
		expected: ABC{"foo", "bar", "qux"},
	}, {
		name:     "ok embedded DEF",
		enum:     enumTypeEmbeddedValue{},
		json:     `{"type": "def", "D": 1, "E": 2, "F": 3}`,
		expected: DEF{1, 2, 3},
	}} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			enum := reflect.New(reflect.TypeOf(tc.enum)).Elem()

			err := enumtag.UnmarshalJSON([]byte(tc.json), enum.Addr().Interface())
			if err != nil && tc.expectedError == "" {
				t.Fatalf("got unexpected error: %v", err)
			}
			if tc.expectedError != "" && (err == nil || !strings.Contains(err.Error(), tc.expectedError)) {
				t.Fatalf("expected error containing %q; got %v", tc.expectedError, err)
			}
			if expected, got := tc.expected, enum.FieldByName("Value").Interface(); !reflect.DeepEqual(expected, got) {
				t.Fatalf("expected %+v; got %+v", expected, got)
			}
			if err != nil {
				return
			}

			reversed, err := enumtag.MarshalJSON(enum.Interface())
			if err != nil {
				t.Fatal(err)
			}

			var expected, got interface{}
			err = json.Unmarshal([]byte(tc.json), &expected)
			if err != nil {
				t.Fatal(err)
			}
			json.Unmarshal(reversed, &got)
			if !reflect.DeepEqual(expected, got) {
				t.Fatalf("expected %+v; got %+v", expected, got)
			}
		})
	}

}
