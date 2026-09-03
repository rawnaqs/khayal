package connections

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeChecker struct {
	response string
	err      error
	calls    int
}

func (f *fakeChecker) GenerateWithSystemTemp(system, user string, temp float64) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if f.response != "" {
		return f.response, nil
	}
	// default: only the debt note contradicts — a false-positive trap for
	// detectors that flag every candidate
	if strings.Contains(user, "owes") {
		return `{"contradicts": true, "because": "payment vs debt"}`, nil
	}
	return `{"contradicts": false}`, nil
}

func contraConn(path, content string, created time.Time) Connection {
	c := simConn(path, created)
	c.Excerpt = content
	return c
}

func TestFindContradictions(t *testing.T) {
	now := time.Now().UTC()
	similar := []Connection{
		contraConn("khayal/old.md", "bob still owes me 100 rupees", now.AddDate(-1, 0, 0)),
		contraConn("khayal/other.md", "unrelated note about go channels", now.AddDate(0, -2, 0)),
	}
	system := "system prompt"

	t.Run("contradicting verdict surfaces with date in label", func(t *testing.T) {
		fc := &fakeChecker{}
		got := findContradictions(context.Background(), fc, system, "bob paid back every rupee today", similar, now)
		if len(got) != 1 || got[0].Type != "contradiction" || got[0].NotePath != "khayal/old.md" {
			t.Fatalf("got %+v", got)
		}
		if !strings.Contains(got[0].Label, "contradicts something you wrote") {
			t.Errorf("label: %q", got[0].Label)
		}
	})

	t.Run("non-contradicting verdicts are dropped", func(t *testing.T) {
		fc := &fakeChecker{response: `{"contradicts": false, "because": "different topics"}`}
		if got := findContradictions(context.Background(), fc, system, "bob paid back every rupee today", similar, now); len(got) != 0 {
			t.Errorf("expected none, got %+v", got)
		}
	})

	t.Run("garbage verdict fails open per candidate", func(t *testing.T) {
		fc := &fakeChecker{response: `I think maybe it does not contradict!`}
		if got := findContradictions(context.Background(), fc, system, "bob paid back every rupee today", similar, now); len(got) != 0 {
			t.Errorf("garbage must be skipped, got %+v", got)
		}
		if fc.calls != len(similar) {
			t.Errorf("expected one call per candidate (%d), got %d", len(similar), fc.calls)
		}
	})

	t.Run("checker errors fail open", func(t *testing.T) {
		fc := &fakeChecker{err: context.DeadlineExceeded}
		if got := findContradictions(context.Background(), fc, system, "bob paid back every rupee today", similar, now); len(got) != 0 {
			t.Errorf("errors must skip candidates, got %+v", got)
		}
	})
}

func TestFindContradictionsNilChecker(t *testing.T) {
	if got := findContradictions(context.Background(), nil, "", "body", nil, time.Now()); len(got) != 0 {
		t.Errorf("nil checker must yield nothing, got %+v", got)
	}
}

type recordingChecker struct {
	fakeChecker
	lastUser string
}

func (r *recordingChecker) GenerateWithSystemTemp(system, user string, temp float64) (string, error) {
	r.lastUser = user
	return r.fakeChecker.GenerateWithSystemTemp(system, user, temp)
}

// Regression: the verdict prompt must contain BOTH sides — the new note's
// own body was silently omitted, leaving the model nothing to contradict.
func TestFindContradictionsPromptIncludesSelfContent(t *testing.T) {
	rc := &recordingChecker{}
	similar := []Connection{
		contraConn("khayal/old.md", "bob still owes me 100 rupees", time.Now().AddDate(-1, 0, 0)),
	}
	findContradictions(context.Background(), rc, "sys", "bob paid back every rupee today", similar, time.Now())
	if !strings.Contains(rc.lastUser, "bob paid back every rupee today") {
		t.Errorf("self content missing from verdict prompt:\n%s", rc.lastUser)
	}
	if !strings.Contains(rc.lastUser, "owes me 100 rupees") {
		t.Errorf("candidate excerpt missing from verdict prompt:\n%s", rc.lastUser)
	}
}
