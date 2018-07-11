package sion

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

func (e *Encoder) encodeArray(rv reflect.Value) error {
	_, err := e.w.Write([]byte{'['})
	if err != nil {
		return err
	}
	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			_, err = e.w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		err = e.encode(rv.Index(i).Interface())
		if err != nil {
			return err
		}
	}
	_, err = e.w.Write([]byte{']'})
	if err != nil {
		return err
	}
	return nil
}

func (e *Encoder) encodeMap(rv reflect.Value) error {
	if rv.Len() == 0 {
		_, err := e.w.Write([]byte("[:]"))
		return err
	}
	_, err := e.w.Write([]byte{'['})
	if err != nil {
		return err
	}
	for i, key := range rv.MapKeys() {
		if i > 0 {
			_, err = e.w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		err = e.encode(key.Interface())
		if err != nil {
			return err
		}
		_, err = e.w.Write([]byte{':'})
		if err != nil {
			return err
		}
		err = e.encode(rv.MapIndex(key).Interface())
		if err != nil {
			return err
		}
	}
	_, err = e.w.Write([]byte{']'})
	if err != nil {
		return err
	}
	return nil
}

func (e *Encoder) encode(v interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(v))
	rk := rv.Type().Kind()
	if rk == reflect.Map {
		return e.encodeMap(rv)
	} else if rk == reflect.Slice || rk == reflect.Array {
		return e.encodeArray(rv)
	} else if rk == reflect.Struct {
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(v)
		if err != nil {
			return err
		}
		s := strings.TrimRight(buf.String(), "\n")
		if len(s) > 1 && s[0] == '{' && s[len(s)-1] == '}' {
			s = s[1 : len(s)-1]
		}
		_, err = e.w.Write([]byte(s))
		return err
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(v)
	if err != nil {
		return err
	}
	s := strings.TrimRight(buf.String(), "\n")
	_, err = e.w.Write([]byte(s))
	return err
}

func (e *Encoder) Encode(v interface{}) error {
	return e.encode(v)
}
