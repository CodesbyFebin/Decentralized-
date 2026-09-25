// Package canon implements the dh/v1 canonical JSON encoding.
//
// Every signed or hashed protocol object is serialized with this encoding so
// that independent implementations produce byte-identical input to the
// signature and hash functions. The rules (see docs/protocol/dh-v1.md §2):
//
//   - objects: keys sorted by byte order, no duplicate keys
//   - no insignificant whitespace
//   - strings: UTF-8; escape `"` and `\`; \b \f \n \r \t use short forms;
//     other code points below U+0020 use \u00xx (lowercase hex); everything
//     else is emitted raw
//   - numbers: integers only, within ±(2^53-1); no fraction, no exponent
//   - true, false, null as usual
package canon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxSafeInt = 1<<53 - 1

// Marshal encodes v as canonical JSON. v is first encoded with encoding/json,
// so struct tags apply; the result is then re-encoded canonically.
func Marshal(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return Canonicalize(raw)
}

// MustMarshal is Marshal for values that are known to be encodable.
func MustMarshal(v any) []byte {
	b, err := Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("canon: %v", err))
	}
	return b
}

// Canonicalize re-encodes arbitrary JSON bytes canonically.
func Canonicalize(raw []byte) ([]byte, error) {
	// encoding/json replaces invalid UTF-8 and unpaired surrogate escapes
	// with U+FFFD; dh/v1 rejects both so every implementation agrees on
	// what was signed.
	if !utf8.Valid(raw) {
		return nil, errors.New("canon: input is not valid UTF-8")
	}
	if err := checkSurrogates(raw); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var tree any
	if err := dec.Decode(&tree); err != nil {
		return nil, fmt.Errorf("canon: decode: %w", err)
	}
	if dec.More() {
		return nil, errors.New("canon: trailing data after JSON value")
	}
	if err := checkDuplicateKeys(raw); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := encode(&buf, tree); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// IsCanonical reports whether raw is already in canonical form.
func IsCanonical(raw []byte) bool {
	c, err := Canonicalize(raw)
	return err == nil && bytes.Equal(c, raw)
}

func encode(buf *bytes.Buffer, v any) error {
	switch t := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if t {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case json.Number:
		s := t.String()
		if strings.ContainsAny(s, ".eE") {
			return fmt.Errorf("canon: non-integer number %q is not allowed", s)
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("canon: number %q: %w", s, err)
		}
		if n > maxSafeInt || n < -maxSafeInt {
			return fmt.Errorf("canon: number %d outside ±(2^53-1)", n)
		}
		buf.WriteString(strconv.FormatInt(n, 10))
	case string:
		return encodeString(buf, t)
	case []any:
		buf.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encode(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeString(buf, k); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := encode(buf, t[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("canon: unsupported type %T", v)
	}
	return nil
}

func encodeString(buf *bytes.Buffer, s string) error {
	if !utf8.ValidString(s) {
		return errors.New("canon: string is not valid UTF-8")
	}
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return nil
}

// checkSurrogates rejects \uXXXX escapes that encode an unpaired UTF-16
// surrogate.
func checkSurrogates(raw []byte) error {
	hex4 := func(i int) (int, bool) {
		if i+4 > len(raw) {
			return 0, false
		}
		v, err := strconv.ParseUint(string(raw[i:i+4]), 16, 32)
		return int(v), err == nil
	}
	inString := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if !inString {
			if c == '"' {
				inString = true
			}
			continue
		}
		switch c {
		case '"':
			inString = false
		case '\\':
			if i+1 < len(raw) && raw[i+1] == 'u' {
				v, ok := hex4(i + 2)
				if !ok {
					return nil // malformed escape; the decoder reports it
				}
				switch {
				case v >= 0xD800 && v <= 0xDBFF:
					if i+11 < len(raw) && raw[i+6] == '\\' && raw[i+7] == 'u' {
						if lo, ok := hex4(i + 8); ok && lo >= 0xDC00 && lo <= 0xDFFF {
							i += 11
							continue
						}
					}
					return errors.New("canon: unpaired UTF-16 surrogate escape")
				case v >= 0xDC00 && v <= 0xDFFF:
					return errors.New("canon: unpaired UTF-16 surrogate escape")
				}
				i += 5
			} else {
				i++
			}
		}
	}
	return nil
}

// checkDuplicateKeys walks the token stream and rejects objects that repeat a
// key. encoding/json silently keeps the last value, which would let two
// parties disagree about what was signed.
func checkDuplicateKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	type frame struct {
		obj    bool
		keys   map[string]struct{}
		expect bool // next string token in an object is a key
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err != nil {
			// io.EOF ends the walk; syntax errors were already reported by
			// the full decode in Canonicalize.
			return nil
		}
		var top *frame
		if len(stack) > 0 {
			top = stack[len(stack)-1]
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				if top != nil && top.obj {
					top.expect = true
				}
				stack = append(stack, &frame{obj: true, keys: map[string]struct{}{}, expect: true})
				continue
			case '[':
				if top != nil && top.obj {
					top.expect = true
				}
				stack = append(stack, &frame{})
				continue
			case '}', ']':
				stack = stack[:len(stack)-1]
				if len(stack) > 0 && stack[len(stack)-1].obj {
					stack[len(stack)-1].expect = true
				}
				continue
			}
		case string:
			if top != nil && top.obj && top.expect {
				if _, dup := top.keys[t]; dup {
					return fmt.Errorf("canon: duplicate object key %q", t)
				}
				top.keys[t] = struct{}{}
				top.expect = false
				continue
			}
		}
		if top != nil && top.obj {
			top.expect = true
		}
	}
}

// Wire encodes v as JSON for transport without HTML escaping, so signed
// payloads travel byte-for-byte as they were signed.
func Wire(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
