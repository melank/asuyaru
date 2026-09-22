package notify

import "testing"

func TestScriptQuotes(t *testing.T) {
	t.Parallel()
	got := script(`日誌 "夜"`, `一行\日記`)
	want := `display notification "一行\\日記" with title "日誌 \"夜\""`
	if got != want {
		t.Fatalf("script = %s", got)
	}
}
