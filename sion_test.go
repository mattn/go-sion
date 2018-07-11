package sion

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func addressOf(v interface{}) interface{} {
	return &v
}

func TestSION(t *testing.T) {
	tests := []struct {
		input  string
		result interface{}
	}{
		{
			input:  `"foo"`,
			result: "foo",
		},
		{
			input:  `"fo\no"`,
			result: "fo\no",
		},
		{
			input:  `314.3`,
			result: 314.3,
		},
		{
			input:  `-314.3`,
			result: -314.3,
		},
		{
			input:  `true`,
			result: true,
		},
		{
			input:  `false`,
			result: false,
		},
		{
			input:  `[true, 1]`,
			result: []interface{}{true, int64(1)},
		},
		{
			input:  `[true, [1: "foo"]]`,
			result: []interface{}{true, map[interface{}]interface{}{int64(1): "foo"}},
		},
		{
			input:  `[true, [:]]`,
			result: []interface{}{true, map[interface{}]interface{}{}},
		},
		{
			input: `
			[
				"nil":      nil,
				"bool":     true,
				"int":      -42,
				"double":   42.195,
				"string":   "漢字、カタカナ、ひらがなの入ったstring😇",
				"array":    [nil, true, 1, 1.0, "one", [1], ["one":1.0]],
				"dictionary":   [
					"nil":nil, "bool":false, "int":0, "double":0.0, "string":"","array":[], "object":[:]
				],
				"url":"https://github.com/dankogai/"
			]
			`,
			result: map[interface{}]interface{}{
				"nil":    nil,
				"bool":   true,
				"int":    int64(-42),
				"double": float64(42.195),
				"string": "漢字、カタカナ、ひらがなの入ったstring😇",
				"array":  []interface{}{nil, true, int64(1), float64(1.0), "one", []interface{}{int64(1)}, map[interface{}]interface{}{"one": float64(1.0)}},
				"dictionary": map[interface{}]interface{}{
					"nil":    nil,
					"bool":   false,
					"int":    int64(0),
					"double": float64(0.0),
					"string": "",
					"array":  []interface{}{},
					"object": map[interface{}]interface{}{},
				},
				"url": "https://github.com/dankogai/",
			},
		},
		{
			input: `
				[
					"array" : [
						nil,
						true,
						1,    // Int in decimal
						1.0,  // Double in decimal
						"one",
						[1],
						["one" : 1.0]
					],
					"bool" : true,
					"data" : .Data("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"),
					"date" : .Date(0x0p+0),
					"dictionary" : [
						"array" : [],
						"bool" : false,
						"double" : 0x0p+0,
						"int" : 0,
						"nil" : nil,
						"object" : [:],
						"string" : ""
					],
					"double" : 0x1.518f5c28f5c29p+5, // Double in hexadecimal
					"int" : -0x2a, // Int in hexadecimal
					"nil" : nil,
					"string" : "漢字、カタカナ、ひらがなの入ったstring😇",
					"url" : "https://github.com/dankogai/",
					true  : "Yes, SION",
					1     : "does accept",
					1.0   : "non-String keys."
				]
				`,
			result: map[interface{}]interface{}{
				"array": []interface{}{nil, true, int64(1), float64(1.0), "one", []interface{}{int64(1)}, map[interface{}]interface{}{"one": float64(1.0)}},
				"bool":  true,
				"data":  []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x01, 0x44, 0x00, 0x3b},
				"date":  time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Local(),
				"dictionary": map[interface{}]interface{}{
					"array":  []interface{}{},
					"bool":   false,
					"double": int64(0.0),
					"int":    int64(0),
					"nil":    nil,
					"object": map[interface{}]interface{}{},
					"string": "",
				},
				"double":     float64(0.0),
				"int":        int64(-0x2a),
				"nil":        nil,
				"string":     "漢字、カタカナ、ひらがなの入ったstring😇",
				"url":        "https://github.com/dankogai/",
				true:         "Yes, SION",
				int64(1):     "does accept",
				float64(1.0): "non-String keys.",
			},
		},
	}

	for _, test := range tests {
		var v interface{}
		err := NewDecoder(strings.NewReader(test.input)).Decode(&v)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v, test.result) {
			t.Fatalf("want %+v but got %+v", test.result, v)
		}
	}
}
