package usage

import (
	"fmt"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// The heatmap and request cells pass borrowed model ids to the clear's
// alias mapper. Reusing the decoded chunk must not change earlier names.
func TestClearedUsageWithBorrowedModels(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	aliases := map[string]string{}
	var chunks []*rowChunk
	for i := range 128 {
		model := fmt.Sprintf("model-%03d", i)
		aliases[model] = fmt.Sprintf("named-%03d", i)
		c := &rowChunk{}
		c.add(Row{Record: Record{Time: now.Add(-time.Minute), Provider: "p", Model: model, Agent: "codex", Input: 1, Status: 200}}, "", 0, false)
		c = c.pack()
		if len(c.Archive) == 0 {
			t.Fatal("fixture was not packed")
		}
		chunks = append(chunks, c)
	}
	clear := &clearSet{alias: provider.UsageAliasIn(aliases), marks: []settings.UsageClear{{Provider: "p", Model: "named-000", At: now.UnixMilli()}}}
	was := clearedAs.Swap(clear)
	t.Cleanup(func() { clearedAs.Store(was) })
	t.Cleanup(ReleaseScratch)
	rows := lentRows()
	defer rows.release()
	ps := &pairedRows{all: chunks}
	var row Row
	visited := 0
	for i, c := range chunks {
		ps.visitChunk(&rows, c, &row, func(_ timedRef, r *Row) {
			visited++
			if kept := (Filter{}).keepsRow(r); kept != (i != 0) {
				t.Fatalf("chunk %d: kept = %v, only model-000 was cleared", i, kept)
			}
			if got, want := clear.alias(r.Model), fmt.Sprintf("named-%03d", i); got != want {
				t.Fatalf("chunk %d counted as %q after decoding over earlier model ids, want %q", i, got, want)
			}
		})
	}
	if visited != len(chunks) {
		t.Fatalf("visited %d packed rows, want %d", visited, len(chunks))
	}
}
