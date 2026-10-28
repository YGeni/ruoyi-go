package cronx

import (
	"testing"
	"time"
)

// RuoYi 自带的三个示例任务（sys_job 种子）用的是 quartz 6 段 + '?'。
func TestIsValidSeedExpressions(t *testing.T) {
	for _, e := range []string{
		"0/10 * * * * ?",
		"0/15 * * * * ?",
		"0/20 * * * * ?",
		"0 0 12 * * ?",
		"0 0/5 14,18 * * ?",
	} {
		if !IsValid(e) {
			t.Errorf("IsValid(%q) = false, want true", e)
		}
	}
	for _, e := range []string{"", "  ", "* * *", "abc", "0/0 * * * * ?", "99 * * * * ?", "0 * * * * L"} {
		if IsValid(e) {
			t.Errorf("IsValid(%q) = true, want false", e)
		}
	}
}

// 每 10 秒：下次触发必须在 10 秒内，且秒位是 0/10/20/30/40/50。
func TestNextEveryTenSeconds(t *testing.T) {
	s, err := Parse("0/10 * * * * ?")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	base := time.Date(2026, 10, 28, 10, 0, 3, 0, time.Local)
	got, err := s.Next(base)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if want := base.Add(7 * time.Second); !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got.Format("15:04:05"), want.Format("15:04:05"))
	}
	if got.Second()%10 != 0 {
		t.Errorf("秒位不是 10 的倍数: %d", got.Second())
	}
	// 连续推进 5 次，每次间隔应为 10 秒
	cur := got
	for i := 0; i < 5; i++ {
		nxt, err := s.Next(cur)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if d := nxt.Sub(cur); d != 10*time.Second {
			t.Errorf("第 %d 次间隔 = %v, want 10s", i+1, d)
		}
		cur = nxt
	}
}

// 每天 12:00:00。
func TestNextDailyNoon(t *testing.T) {
	s, err := Parse("0 0 12 * * ?")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	base := time.Date(2026, 10, 28, 13, 30, 0, 0, time.Local)
	got, err := s.Next(base)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	want := time.Date(2026, 10, 29, 12, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}
}

// 周位指定：每周一（dow=1）09:00。日位是 ? -> 只按周判定。
func TestNextWeekly(t *testing.T) {
	s, err := Parse("0 0 9 ? * 1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// 2026-10-28 是周三
	base := time.Date(2026, 10, 28, 10, 0, 0, 0, time.Local)
	got, err := s.Next(base)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if got.Weekday() != time.Monday {
		t.Errorf("下次触发是 %v, want 周一", got.Weekday())
	}
}

// 2 月 30 日永不触发 -> 返回错误而不是死循环。
func TestNextImpossible(t *testing.T) {
	s, err := Parse("0 0 0 30 2 ?")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := s.Next(time.Now()); err == nil {
		t.Error("永不触发的表达式应返回 error")
	}
}
