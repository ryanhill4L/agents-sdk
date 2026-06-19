package schedules

import (
	"testing"
	"time"
)

func mustCompile(t *testing.T, expr string) Schedule {
	t.Helper()
	s := Schedule{Cron: expr}
	if err := s.compile(); err != nil {
		t.Fatalf("compile(%q): %v", expr, err)
	}
	return s
}

func TestCronMatches(t *testing.T) {
	// 09:00 on weekdays. 2024-01-01 is a Monday.
	s := mustCompile(t, "0 9 * * 1-5")

	monday9 := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	if !s.Matches(monday9) {
		t.Error("expected match at Monday 09:00")
	}
	if s.Matches(monday9.Add(time.Minute)) {
		t.Error("did not expect match at 09:01")
	}

	saturday9 := time.Date(2024, 1, 6, 9, 0, 0, 0, time.UTC)
	if s.Matches(saturday9) {
		t.Error("did not expect match on Saturday")
	}
}

func TestCronStep(t *testing.T) {
	s := mustCompile(t, "*/15 * * * *")
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, m := range []int{0, 15, 30, 45} {
		if !s.Matches(base.Add(time.Duration(m) * time.Minute)) {
			t.Errorf("expected match at minute %d", m)
		}
	}
	if s.Matches(base.Add(7 * time.Minute)) {
		t.Error("did not expect match at minute 7")
	}
}

func TestCronList(t *testing.T) {
	s := mustCompile(t, "0,30 12 * * *")
	noon := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	if !s.Matches(noon) || !s.Matches(noon.Add(30*time.Minute)) {
		t.Error("expected matches at 12:00 and 12:30")
	}
	if s.Matches(noon.Add(15 * time.Minute)) {
		t.Error("did not expect match at 12:15")
	}
}

func TestCronSundayBothForms(t *testing.T) {
	sunday := time.Date(2024, 1, 7, 9, 0, 0, 0, time.UTC) // a Sunday
	for _, expr := range []string{"0 9 * * 0", "0 9 * * 7"} {
		s := mustCompile(t, expr)
		if !s.Matches(sunday) {
			t.Errorf("%q should match Sunday 09:00", expr)
		}
	}
}

func TestCronInvalid(t *testing.T) {
	for _, expr := range []string{"", "1 2 3", "60 * * * *", "* * * * 9", "*/0 * * * *"} {
		s := Schedule{Cron: expr}
		if err := s.compile(); err == nil {
			t.Errorf("expected error for %q", expr)
		}
	}
}
