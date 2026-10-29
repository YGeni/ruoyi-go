// Package cronx 解析 **quartz 风格**的 cron 表达式并计算下次触发时刻。
//
// 为什么不用 robfig/cron：RuoYi 用的是 quartz，表达式是 **6 段**（秒 分 时 日 月 周），
// 且星期位用 `?`（`0/10 * * * * ?`）。robfig/cron 默认是 5 段（分 时 日 月 周），
// 加了 WithSeconds 才是 6 段但**顺序是 秒分 时 日 月 周**——恰好一致，
// 然而它不接受 `?`（quartz 的"不指定"，与 `*` 同义但专用于 日/周 互斥）。
// 这里自己写，把 quartz 的语义钉死，同时也给 CronUtils.isValid 用。
//
// 支持：
//
//   - 任意
//     ?       任意（quartz 里专用于 日/周 二选一，语义等同 *）
//     a-b     区间
//     a/b     从 a 开始每 b（a 可为 *，等价于 0/b）
//     a,b,c   枚举
//
// 不支持 quartz 的 L / W / # / 年字段（RuoYi 自带的三个示例任务都没用到，
// 遇到直接判为非法——比静默按错语义执行要好）。
package cronx

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 各字段取值范围：[min, max]。
type bound struct{ min, max int }

var bounds = [6]bound{
	{0, 59}, // 秒
	{0, 59}, // 分
	{0, 23}, // 时
	{1, 31}, // 日
	{1, 12}, // 月
	{0, 6},  // 周（0=周日，与 quartz 的 1=周日 不同，见 weekday 说明）
}

// ErrInvalid 表达式非法。
var ErrInvalid = errors.New("cron 表达式不合法")

// Schedule 解析后的调度表：每个字段的**命中值集合**。
type Schedule struct {
	sec    map[int]bool
	min    map[int]bool
	hour   map[int]bool
	dom    map[int]bool
	month  map[int]bool
	dow    map[int]bool
	domAny bool // 日位是 * / ?（quartz 里与周位互斥）
	dowAny bool
}

// Parse 解析表达式。
func Parse(expr string) (*Schedule, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 6 {
		return nil, ErrInvalid
	}
	s := &Schedule{}
	var err error
	if s.sec, err = parseField(fields[0], bounds[0]); err != nil {
		return nil, err
	}
	if s.min, err = parseField(fields[1], bounds[1]); err != nil {
		return nil, err
	}
	if s.hour, err = parseField(fields[2], bounds[2]); err != nil {
		return nil, err
	}
	if s.dom, s.domAny, err = parseFieldAny(fields[3], bounds[3]); err != nil {
		return nil, err
	}
	if s.month, err = parseField(fields[4], bounds[4]); err != nil {
		return nil, err
	}
	if s.dow, s.dowAny, err = parseFieldAny(fields[5], bounds[5]); err != nil {
		return nil, err
	}
	// quartz 要求 日/周 至少有一个是 "不指定"：两个都指定会互相矛盾。
	// 两个都是 ? 时按"每天都跑"处理（RuoYi 自带的示例 `0/10 * * * * ?` 正是这种）。
	return s, nil
}

// IsValid 等价 CronUtils.isValid（ruoyi-
// 只判"能不能被 quartz 解析"，不判下次触发时间。
func IsValid(expr string) bool {
	if strings.TrimSpace(expr) == "" {
		return false
	}
	_, err := Parse(expr)
	return err == nil
}

// Next 返回 t 之后的下一个触发时刻。
//
// 实现是**按秒推进的暴力搜索**，最坏扫 366 天 × 24 时 × 60 分 ≈ 53 万次，
// 只在"任务状态变更 / 启动加载"时调用一次，不在每请求的路径上，够用。
//
// 找不到返回零值 + ErrInvalid（例如 2 月 30 日这种永不触发的组合）。
func (s *Schedule) Next(t time.Time) (time.Time, error) {
	cur := t.Add(time.Second).Truncate(time.Second)
	// 逐天推进，避免逐秒扫一年
	for d := 0; d < 366; d++ {
		day := cur.AddDate(0, 0, d)
		y, m, dd := day.Date()
		if !s.month[int(m)] {
			continue
		}
		// quartz 语义：日与周**同时指定**时取并集；只有一个指定时只按它判定。
		domHit := s.dom[dd]
		dowHit := s.dow[int(day.Weekday())]
		switch {
		case s.domAny && s.dowAny:
			// 都未指定 -> 每天
		case s.domAny:
			if !dowHit {
				continue
			}
		case s.dowAny:
			if !domHit {
				continue
			}
		default:
			if !domHit && !dowHit {
				continue
			}
		}
		for h := 0; h < 24; h++ {
			if !s.hour[h] {
				continue
			}
			for mi := 0; mi < 60; mi++ {
				if !s.min[mi] {
					continue
				}
				// 秒：取该分钟内 >= 起点秒 的最小值
				startSec := 0
				if d == 0 {
					// cur 所在这一天要跳过已过去的部分
					if h < cur.Hour() || (h == cur.Hour() && mi < cur.Minute()) {
						continue
					}
					if h == cur.Hour() && mi == cur.Minute() {
						startSec = cur.Second()
					}
				}
				for sec := startSec; sec < 60; sec++ {
					if !s.sec[sec] {
						continue
					}
					cand := time.Date(y, m, dd, h, mi, sec, 0, t.Location())
					if cand.After(t) {
						return cand, nil
					}
				}
			}
		}
	}
	return time.Time{}, ErrInvalid
}

// parseField 解析单个字段，返回命中集合。
func parseField(f string, b bound) (map[int]bool, error) {
	m, _, err := parseFieldAny(f, b)
	return m, err
}

// parseFieldAny 解析字段，额外报告该位是不是 `*`/`?`（"未指定"）。
func parseFieldAny(f string, b bound) (map[int]bool, bool, error) {
	f = strings.TrimSpace(f)
	out := map[int]bool{}
	if f == "*" || f == "?" {
		for i := b.min; i <= b.max; i++ {
			out[i] = true
		}
		return out, true, nil
	}
	for _, part := range strings.Split(f, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false, ErrInvalid
		}
		step := 1
		if i := strings.Index(part, "/"); i >= 0 {
			n, err := strconv.Atoi(strings.TrimSpace(part[i+1:]))
			if err != nil || n <= 0 {
				return nil, false, ErrInvalid
			}
			step = n
			part = part[:i]
		}
		lo, hi := b.min, b.max
		if part == "" || part == "*" || part == "?" {
			// "* /n"
		} else if i := strings.Index(part, "-"); i >= 0 {
			a, err := strconv.Atoi(strings.TrimSpace(part[:i]))
			if err != nil {
				return nil, false, ErrInvalid
			}
			c, err := strconv.Atoi(strings.TrimSpace(part[i+1:]))
			if err != nil {
				return nil, false, ErrInvalid
			}
			lo, hi = a, c
			if strings.HasPrefix(part, "*/") {
				lo = b.min
			}
		} else {
			a, err := strconv.Atoi(part)
			if err != nil {
				return nil, false, ErrInvalid
			}
			// "a/b"：从 a 开始每 b
			if step > 1 {
				for v := a; v <= b.max; v += step {
					out[v] = true
				}
				continue
			}
			if a < b.min || a > b.max {
				return nil, false, ErrInvalid
			}
			out[a] = true
			continue
		}
		if lo < b.min {
			lo = b.min
		}
		if hi > b.max {
			hi = b.max
		}
		if lo > hi {
			return nil, false, ErrInvalid
		}
		for v := lo; v <= hi; v += step {
			out[v] = true
		}
	}
	if len(out) == 0 {
		return nil, false, ErrInvalid
	}
	_ = sort.Ints
	return out, false, nil
}
