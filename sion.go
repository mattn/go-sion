package sion

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type Decoder struct {
	r *bufio.Reader
	v reflect.Value
}

type Array []interface{}
type Map map[interface{}]interface{}

func (m Map) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.WriteString("{")
	if err != nil {
		return nil, err
	}
	for k, v := range m {
		var vbuf bytes.Buffer
		err = json.NewEncoder(&vbuf).Encode(v)
		if err != nil {
			return nil, err
		}
		if buf.Len() > 1 {
			_, err = buf.WriteString(",")
			if err != nil {
				return nil, err
			}
		}
		fmt.Fprintf(&buf, "%q:%s", fmt.Sprint(k), strings.TrimRight(vbuf.String(), "\n"))
	}
	_, err = buf.WriteString("}")
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hashable(v interface{}) bool {
	if v == nil {
		return false
	}
	k := reflect.TypeOf(v).Kind()
	return k != reflect.Map && k != reflect.Slice
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: bufio.NewReader(r),
	}
}

func (d *Decoder) skipWhite() error {
	for {
		b, err := d.r.Peek(1)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		r := rune(b[0])
		if r == '/' {
			_, err = d.r.Discard(1)
			if err != nil {
				return err
			}
			b, err = d.r.Peek(1)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				return err
			}
			if b[0] != '/' {
				return nil
			}
			_, _, err = d.r.ReadLine()
			continue
		}
		if r != '\n' && r != '\r' && r != '\t' && r != ' ' {
			return nil
		}
		_, err = d.r.Discard(1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Decoder) readInt(buf *bytes.Buffer) error {
	for {
		r, _, err := d.r.ReadRune()
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				break
			}
			return err
		}
		if r < '0' || '9' < r {
			break
		}
		buf.WriteRune(r)
	}
	return nil
}

func (d *Decoder) decodeString() (string, error) {
	r, _, err := d.r.ReadRune()
	if err != nil {
		return "", err
	}
	if r != '"' {
		return "", errors.New("parse string error")
	}
	var buf bytes.Buffer
	for {
		r, _, err := d.r.ReadRune()
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				break
			}
			return "", err
		}
		if r == '\\' {
			r, _, err = d.r.ReadRune()
			if err != nil {
				return "", err
			}
			switch r {
			case 't':
				r = '\t'
			case '\\':
				r = '\\'
			case 'b':
				r = '\b'
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			}
		} else if unicode.IsControl(r) {
			return "", errors.New("parse string error")
		} else if r == '"' {
			break
		}
		buf.WriteRune(r)
	}
	return buf.String(), nil
}

func (d *Decoder) decodeNil() (interface{}, error) {
	var buf bytes.Buffer
	for {
		r, _, err := d.r.ReadRune()
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				break
			}
			return false, err
		}
		if r != 'n' && r != 'i' && r != 'l' {
			err = d.r.UnreadRune()
			if err != nil {
				return false, err
			}
			break
		}
		buf.WriteRune(r)
	}
	if buf.String() == "nil" {
		return nil, nil
	}
	return false, errors.New("parse nil error")
}

func (d *Decoder) decodeBool() (bool, error) {
	var buf bytes.Buffer
	for {
		r, _, err := d.r.ReadRune()
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				break
			}
			return false, err
		}
		if !unicode.IsLetter(r) {
			err = d.r.UnreadRune()
			if err != nil {
				return false, err
			}
			break
		}
		buf.WriteRune(r)
	}
	s := buf.String()
	if s == "true" {
		return true, nil
	} else if s == "false" {
		return false, nil
	}
	return false, errors.New("parse bool error")
}

func (d *Decoder) decodeNumber() (interface{}, error) {
	dot := false
	var buf bytes.Buffer
loop:
	for {
		r, _, err := d.r.ReadRune()
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				break
			}
			return nil, err
		}
		switch r {
		case '.':
			dot = true
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		case 'a', 'b', 'c', 'd', 'e', 'f', 'x', '+', 'p':
		default:
			err = d.r.UnreadRune()
			if err != nil {
				return false, err
			}
			break loop
		}
		buf.WriteRune(r)
	}
	s := buf.String()
	var err error
	if !dot {
		var i int64
		_, err = fmt.Sscan(s, &i)
		if err != nil {
			return nil, err
		}
		return i, nil
	}
	var f float64
	_, err = fmt.Sscan(s, &f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (d *Decoder) decodeArrayOrObject() (interface{}, error) {
	_, _, err := d.r.ReadRune()
	if err != nil {
		return nil, err
	}
	err = d.skipWhite()
	if err != nil {
		return nil, err
	}
	b, err := d.r.Peek(1)
	if err != nil {
		return nil, err
	}
	if b[0] == ':' {
		_, _, err := d.r.ReadRune()
		if err != nil {
			return nil, err
		}
		err = d.skipWhite()
		if err != nil {
			return nil, err
		}
		r, _, err := d.r.ReadRune()
		if err != nil {
			return nil, err
		}
		if r != ']' {
			return nil, errors.New("parse error: invalid empty map")
		}
		return Map{}, nil
	} else if b[0] == ']' {
		_, _, err := d.r.ReadRune()
		if err != nil {
			return nil, err
		}
		return Array{}, nil
	}
	k, err := d.decodeAny()
	if err != nil {
		return nil, err
	}
	d.skipWhite()
	if err != nil {
		return nil, err
	}
	r, _, err := d.r.ReadRune()
	if err != nil {
		return nil, err
	}
	switch r {
	case ']':
		return Array{k}, nil
	case ',':
		ret := Array{k}
		for {
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			k, err = d.decodeAny()
			if err != nil {
				return nil, err
			}
			ret = append(ret, k)
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			r, _, err := d.r.ReadRune()
			if err != nil {
				return nil, err
			}
			if r == ']' {
				break
			}
			if r != ',' {
				return nil, errors.New("parse error: invalid array")
			}
		}
		return ret, nil
	case ':':
		err = d.skipWhite()
		if err != nil {
			return nil, err
		}
		v, err := d.decodeAny()
		if err != nil {
			return nil, err
		}
		ret := Map{}
		if hashable(k) {
			ret[k] = v
		} else {
			ret[&k] = v
		}
		for {
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			r, _, err = d.r.ReadRune()
			if err != nil {
				return nil, err
			}
			if r == ']' {
				break
			}
			if r != ',' {
				return nil, errors.New("parse error: invalid map")
			}
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			k, err := d.decodeAny()
			if err != nil {
				return nil, err
			}
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			r, _, err := d.r.ReadRune()
			if err != nil {
				return nil, err
			}
			if r != ':' {
				return nil, errors.New("parse error: invalid map")
			}
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			v, err := d.decodeAny()
			if err != nil {
				return nil, err
			}
			if hashable(k) {
				ret[k] = v
			} else {
				ret[&k] = v
			}
		}
		return ret, nil
	}
	return nil, nil
}

func (d *Decoder) decodeAny() (interface{}, error) {
	b, err := d.r.Peek(1)
	if err != nil {
		return nil, err
	}
	switch b[0] {
	case '[':
		return d.decodeArrayOrObject()
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return d.decodeNumber()
	case '"':
		return d.decodeString()
	case 't', 'f':
		return d.decodeBool()
	case 'n':
		return d.decodeNil()
	case '.':
		s, err := d.r.ReadString('(')
		if err != nil {
			return nil, err
		}
		switch s {
		case ".Data(":
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			v, err := d.decodeString()
			if err != nil {
				return nil, err
			}
			b, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, err
			}
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			r, _, err := d.r.ReadRune()
			if err != nil {
				return nil, err
			}
			if r != ')' {
				return nil, errors.New("parse Data error")
			}
			return b, nil
		case ".Date(":
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			v, err := d.decodeNumber()
			if err != nil {
				return nil, err
			}
			var t time.Time
			switch vv := v.(type) {
			case float64:
				t = time.Unix(int64(vv), 0)
			case int64:
				t = time.Unix(vv, 0)
			}
			err = d.skipWhite()
			if err != nil {
				return nil, err
			}
			r, _, err := d.r.ReadRune()
			if err != nil {
				return nil, err
			}
			if r != ')' {
				return nil, errors.New("parse Date error")
			}
			return t, nil
		}
	}
	return nil, errors.New("parse error: invalid letter: " + string(b))
}

func (d *Decoder) Decode(ref interface{}) error {
	err := d.skipWhite()
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return err
	}
	v, err := d.decodeAny()
	if err != nil {
		return err
	}
	switch vv := ref.(type) {
	case *interface{}:
		*vv = v
	case *int64:
		*vv = v.(int64)
	case *float64:
		*vv = v.(float64)
	case *string:
		*vv = v.(string)
	case *bool:
		*vv = v.(bool)
	default:
		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(v)
		if err != nil {
			return fmt.Errorf("marshal error: %v", err)
		}
		return json.NewDecoder(&buf).Decode(ref)
	}
	for {
		r, _, err := d.r.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if !unicode.IsSpace(r) {
			return errors.New("parse error")
		}
	}
	return nil
}
