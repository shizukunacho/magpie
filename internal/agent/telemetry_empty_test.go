package agent

import (
	"os"
	"testing"
)

// Opening the menu bar polls Crush's telemetry more than once. An empty
// config, or JSON null, has no switch set and must remain readable (#6197).
func TestCrushTelemetryEmptyConfig(t *testing.T) {
	for _, input := range []struct{ name, body string }{
		{"empty", ""}, {"whitespace", " \n\t"}, {"null", "null"},
	} {
		t.Run(input.name, func(t *testing.T) {
			home, path := telemetryHome(t, ".config/crush/crush.json", "{}")
			if err := os.WriteFile(path, []byte(input.body), 0o600); err != nil {
				t.Fatal(err)
			}
			switcher := crushTelemetry(here(home), path)
			for range 3 {
				if st, err := switcher.Get(); err != nil || st.Off || len(st.Own) != 0 {
					t.Fatalf("empty config has a telemetry setting: %+v, %v", st, err)
				}
			}
			if got := readFile(path); got != input.body {
				t.Fatalf("reading telemetry changed the config: %q", got)
			}
		})
	}
}
