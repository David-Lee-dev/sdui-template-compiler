package sduicompiler

import (
	"math"
	"strconv"
	"strings"
)

// Value is a JSON-compatible value: nil, bool, int64, float64, string,
// []Value, or *Object. Objects are ordered so serialization reproduces the
// key order of the YAML source, matching the JS reference implementation.
type Value any

// Object is a JSON object that preserves key insertion order, mirroring the
// property-order semantics of ECMAScript objects the golden fixtures encode.
type Object struct {
	keys  []string
	items map[string]Value
}

// NewObject returns an empty ordered object.
func NewObject() *Object {
	return &Object{items: map[string]Value{}}
}

// Set stores a value under key. A new key is appended; an existing key keeps
// its original position (ECMAScript assignment semantics).
func (o *Object) Set(key string, value Value) {
	if _, ok := o.items[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.items[key] = value
}

// Get returns the value for key and whether it exists.
func (o *Object) Get(key string) (Value, bool) {
	v, ok := o.items[key]
	return v, ok
}

// Has reports whether key exists.
func (o *Object) Has(key string) bool {
	_, ok := o.items[key]
	return ok
}

// Keys returns the keys in insertion order.
func (o *Object) Keys() []string { return o.keys }

// Len returns the number of entries.
func (o *Object) Len() int { return len(o.keys) }

// CompactJSON serializes a value exactly like JSON.stringify(value): compact,
// UTF-8, non-ASCII unescaped, ECMAScript number formatting.
func CompactJSON(v Value) string {
	var sb strings.Builder
	writeJSON(&sb, v, "", "")
	return sb.String()
}

// PrettyJSON serializes a value exactly like JSON.stringify(value, null, 2).
func PrettyJSON(v Value) string {
	var sb strings.Builder
	writeJSON(&sb, v, "  ", "")
	return sb.String()
}

func writeJSON(sb *strings.Builder, v Value, indent, current string) {
	switch t := v.(type) {
	case nil:
		sb.WriteString("null")
	case bool:
		if t {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case int64:
		sb.WriteString(strconv.FormatInt(t, 10))
	case float64:
		sb.WriteString(formatJSNumber(t))
	case string:
		writeJSONString(sb, t)
	case []Value:
		if len(t) == 0 {
			sb.WriteString("[]")
			return
		}
		inner := current + indent
		sb.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				sb.WriteByte(',')
			}
			if indent != "" {
				sb.WriteByte('\n')
				sb.WriteString(inner)
			}
			writeJSON(sb, item, indent, inner)
		}
		if indent != "" {
			sb.WriteByte('\n')
			sb.WriteString(current)
		}
		sb.WriteByte(']')
	case *Object:
		if t.Len() == 0 {
			sb.WriteString("{}")
			return
		}
		inner := current + indent
		sb.WriteByte('{')
		for i, key := range t.keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			if indent != "" {
				sb.WriteByte('\n')
				sb.WriteString(inner)
			}
			writeJSONString(sb, key)
			sb.WriteByte(':')
			if indent != "" {
				sb.WriteByte(' ')
			}
			writeJSON(sb, t.items[key], indent, inner)
		}
		if indent != "" {
			sb.WriteByte('\n')
			sb.WriteString(current)
		}
		sb.WriteByte('}')
	default:
		panic("unsupported Value type in JSON serializer")
	}
}

// writeJSONString escapes like JSON.stringify: only `"`, `\`, and control
// characters — non-ASCII passes through as raw UTF-8.
func writeJSONString(sb *strings.Builder, s string) {
	const hex = "0123456789abcdef"
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		case '\r':
			sb.WriteString(`\r`)
		case '\b':
			sb.WriteString(`\b`)
		case '\f':
			sb.WriteString(`\f`)
		default:
			if r < 0x20 {
				sb.WriteString(`\u00`)
				sb.WriteByte(hex[r>>4])
				sb.WriteByte(hex[r&0xf])
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
}

// formatJSNumber renders a float64 like ECMAScript Number#toString: integral
// values below 1e21 print without a fractional part, decimal notation is used
// for 1e-6 <= |x| < 1e21, exponential notation otherwise with a bare exponent.
func formatJSNumber(f float64) string {
	if f == 0 {
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		// Go pads the exponent to two digits ("1e-05"); JS does not ("1e-5").
		if i := strings.LastIndexByte(s, 'e'); i >= 0 {
			mantissa, exp := s[:i], s[i+1:]
			sign := ""
			if exp[0] == '+' || exp[0] == '-' {
				sign, exp = string(exp[0]), exp[1:]
			}
			exp = strings.TrimLeft(exp, "0")
			if exp == "" {
				exp = "0"
			}
			s = mantissa + "e" + sign + exp
		}
		return s
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
