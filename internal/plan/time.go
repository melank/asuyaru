// Package plan は予定、TODO、定期、時間割の規則を持つ。
package plan

import (
	"fmt"
	"time"
)

// Clock は Mac のローカルの時と分である。
type Clock struct {
	Hour   int
	Minute int
}

// FormatDate は t を Mac のローカル日付 YYYY-MM-DD にする。
func FormatDate(t time.Time) string {
	local := t.In(time.Local)
	return fmt.Sprintf("%04d-%02d-%02d", local.Year(), int(local.Month()), local.Day())
}

// ClockFrom は t の Mac のローカルの時と分を返す。
func ClockFrom(t time.Time) Clock {
	local := t.In(time.Local)
	return Clock{Hour: local.Hour(), Minute: local.Minute()}
}
