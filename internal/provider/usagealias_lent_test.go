package provider

import (
	"fmt"
	"testing"
	"unsafe"
)

// Usage's lentRows reuses the bytes behind a model id after visiting a
// chunk. Both caches must own the ids they keep, including unchanged ids.
func TestUsageAliasKeepsBorrowedModels(t *testing.T) {
	for _, kind := range []string{"aliases", "groups"} {
		t.Run(kind, func(t *testing.T) {
			aliases, groups := map[string]string{}, map[string]string{}
			var inputs, wants []string
			for i := range 128 {
				model := fmt.Sprintf("model-%03d", i)
				if kind == "aliases" {
					aliases[model] = "named-" + model
					inputs, wants = append(inputs, model), append(wants, aliases[model])
				} else {
					groups["auto-"+model] = model
					inputs, wants = append(inputs, "group/auto-"+model), append(wants, "auto/"+model)
				}
			}
			t.Run("mapped", func(t *testing.T) {
				alias := usageAliasFrom(aliases, groups)
				buf := make([]byte, len(inputs[0]))
				for i, input := range inputs {
					copy(buf, input)
					if got := alias(unsafe.String(unsafe.SliceData(buf), len(buf))); got != wants[i] {
						t.Fatalf("after reusing the row buffer, %s counted as %q, want %q", input, got, wants[i])
					}
				}
				for i, input := range inputs {
					if got := alias(input); got != wants[i] {
						t.Fatalf("cached %s counted as %q, want %q", input, got, wants[i])
					}
				}
			})
			t.Run("unchanged", func(t *testing.T) {
				alias := usageAliasFrom(aliases, groups)
				buf := []byte("unmapped-one")
				alias(unsafe.String(unsafe.SliceData(buf), len(buf)))
				cached := alias("unmapped-one")
				copy(buf, "unmapped-two")
				if cached != "unmapped-one" {
					t.Fatalf("cached unchanged model changed with the row buffer: %q", cached)
				}
			})
		})
	}
}
