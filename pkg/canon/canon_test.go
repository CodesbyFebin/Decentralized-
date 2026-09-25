package canon

import "testing"

func TestCanonicalOrderingAndEscapes(t *testing.T) {
	got, err := Canonicalize([]byte(`{ "b": 1, "a": [true, null, "x\n\u0001é"], "c": {"z": -5, "y": "<&>"} }`))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":[true,null,"x\n\u0001é"],"b":1,"c":{"y":"<&>","z":-5}}`
	if string(got) != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
	if !IsCanonical(got) {
		t.Fatal("canonical output is not a fixed point")
	}
}

func TestRejectsFloatsAndLargeInts(t *testing.T) {
	for _, in := range []string{`{"a":1.5}`, `{"a":1e3}`, `{"a":9007199254740992}`} {
		if _, err := Canonicalize([]byte(in)); err == nil {
			t.Fatalf("%s: expected error", in)
		}
	}
}

func TestRejectsDuplicateKeys(t *testing.T) {
	for _, in := range []string{`{"a":1,"a":2}`, `{"x":{"a":1,"b":[{"c":1,"c":2}]}}`} {
		if _, err := Canonicalize([]byte(in)); err == nil {
			t.Fatalf("%s: expected duplicate-key error", in)
		}
	}
	if _, err := Canonicalize([]byte(`{"a":{"a":1},"b":[{"a":1},{"a":2}]}`)); err != nil {
		t.Fatalf("same key in sibling objects must be allowed: %v", err)
	}
}

func TestMarshalStruct(t *testing.T) {
	type s struct {
		Z string `json:"z"`
		A int64  `json:"a"`
	}
	got := string(MustMarshal(s{Z: "q", A: 7}))
	if got != `{"a":7,"z":"q"}` {
		t.Fatal(got)
	}
}

func TestRejectsLoneSurrogatesAndBadUTF8(t *testing.T) {
	bs := byte('\\')
	esc := func(h string) []byte { return append([]byte{'"', bs, 'u'}, append([]byte(h), '"')...) }
	for _, raw := range [][]byte{esc("d800"), esc("dc00"), {'"', 0xff, '"'}} {
		if _, err := Canonicalize(raw); err == nil {
			t.Fatalf("%q accepted", raw)
		}
	}
	pair := []byte{'"', bs, 'u', 'd', '8', '3', 'd', bs, 'u', 'd', 'e', '0', '0', '"'}
	got, err := Canonicalize(pair)
	if err != nil || string(got) != "\"\U0001F600\"" {
		t.Fatalf("pair: %q %v", got, err)
	}
}
