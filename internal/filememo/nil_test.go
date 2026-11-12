package filememo

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

// A successful parse can be nil, as for an empty telemetry config or JSON
// null (#6197). Every way of reusing it must return that same value.
func TestReadNil(t *testing.T) {
	for _, read := range []struct {
		name string
		do   func(string, string, func([]byte) (any, error)) (any, error)
	}{{"Read", Read[any]}, {"ReadNow", ReadNow[any]}} {
		for _, state := range []string{"recent", "settled", "held"} {
			for _, input := range []struct{ name, body string }{
				{"empty", ""}, {"whitespace", " \n\t"}, {"null", "null"},
			} {
				t.Run(read.name+"/"+state+"/"+input.name, func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						p := filepath.Join(t.TempDir(), "config.json")
						stamp := time.Now().Add(-time.Minute)
						if state == "recent" {
							stamp = time.Now()
						}
						write := func(body string, at time.Time) {
							t.Helper()
							if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
								t.Fatal(err)
							}
							if err := os.Chtimes(p, at, at); err != nil {
								t.Fatal(err)
							}
						}
						write(input.body, stamp)
						if state == "held" {
							defer Hold()()
						}
						n := 0
						parse := func(b []byte) (any, error) {
							n++
							if len(bytes.TrimSpace(b)) == 0 {
								return nil, nil
							}
							var v any
							err := json.Unmarshal(b, &v)
							return v, err
						}
						for range 3 {
							if v, err := read.do("nil", p, parse); err != nil || v != nil {
								t.Fatalf("empty config: %v, %v", v, err)
							}
						}
						if n != 1 {
							t.Fatalf("parsed an unchanged empty config %d times", n)
						}
						if state == "settled" && input.name == "null" {
							write("", stamp) // truncated in the same file-system tick
							if v, err := read.do("nil", p, parse); err != nil || v != nil {
								t.Fatalf("truncated empty config: %v, %v", v, err)
							}
						}
						beforeChanged := n
						write(`"changed"`, stamp.Add(time.Second))
						Look(p)
						if v, err := read.do("nil", p, parse); err != nil || v != "changed" || n != beforeChanged+1 {
							t.Fatalf("after a change: %v, %v, parsed %d times", v, err, n)
						}
						write(`{"unfinished":`, stamp.Add(2*time.Second))
						Look(p)
						if _, err := read.do("nil", p, parse); err == nil {
							t.Fatal("a parse error became a successful empty value")
						}
					})
				})
			}
		}
	}
}

func TestReadTypedNil(t *testing.T) {
	p := filepath.Join(t.TempDir(), "null.json")
	if err := os.WriteFile(p, []byte("null"), 0o600); err != nil {
		t.Fatal(err)
	}
	parse := func(b []byte) (*int, error) {
		var v *int
		err := json.Unmarshal(b, &v)
		return v, err
	}
	for range 3 {
		if v, err := Read("nil pointer", p, parse); err != nil || v != nil {
			t.Fatalf("nil pointer: %v, %v", v, err)
		}
		v, err := Read("pointer in any", p, func(b []byte) (any, error) { return parse(b) })
		if ptr, ok := v.(*int); err != nil || !ok || ptr != nil {
			t.Fatalf("nil pointer lost its type: %#v, %v", v, err)
		}
	}
}
