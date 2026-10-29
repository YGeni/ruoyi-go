// Package sqlstats 用 GORM 回调做 SQL 执行统计，补上 druid 的"SQL 监控"。
//
// DIFF(review) —— 为什么需要这个包：
//
// Java 的 druid 既是连接池、又在 JDBC 驱动层包了一层，所以每条 SQL 都从它
// 那儿过，它顺手就记下执行次数/耗时/最慢/错误数 —— **天生站在那个位置上**。
// Go 侧 database/sql 只管拿连接、执行、还连接，gorm 只管拼 SQL 并执行，
// 两层都不做统计，所以 SQL 从它们中间过去时**没人记**。
//
// 这里用 gorm 的 Callback 在每条 SQL 前后打点，把"druid 站在驱动层"这件事
// 补回来。与 druid 的差异写在 Stat 的注释里。
package sqlstats

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// maxEntries SQL 模板的最大保留条数。
//
// 按模板聚合后条数取决于代码里有多少种不同形状的 SQL，正常情况下几十条；
// 但如果某处拼接了动态 SQL（比如 IN 后面跟不定数量的占位符没被折叠），
// 会无限增长，所以必须有上限。满了就淘汰**最久没执行**的那条（近似 LRU）。
// MaxEntries 保留上限，页面会显示这个值。
const MaxEntries = 500

// Stat 一条 SQL 模板的累计统计，对应 druid SQL 监控表里的一行。
type Stat struct {
	SQL      string        // 归一化后的 SQL 模板（字面量换成 ?）
	Count    int64         // 执行次数
	Total    time.Duration // 累计耗时
	Max      time.Duration // 最慢一次
	Min      time.Duration // 最快一次
	Errors   int64         // 失败次数
	Rows     int64         // 累计影响/返回行数
	LastAt   time.Time     // 最后执行时间
	LastTook time.Duration // 最后一次耗时
}

// Avg 平均耗时。
func (s Stat) Avg() time.Duration {
	if s.Count == 0 {
		return 0
	}
	return s.Total / time.Duration(s.Count)
}

type entry struct {
	Stat
}

var (
	mu    sync.RWMutex
	stats = map[string]*entry{}
	on    bool // 是否已安装
)

// Install 给指定的 *gorm.DB 装上统计回调。
//
// 时机：必须在业务查询之前。放在 pkg/db.Init 里，建连成功后立刻调用。
// 重复调用无害（gorm 的 Register 同名回调是替换语义）。
// Install 给指定的 *gorm.DB 装上统计回调。
//
// 时机：必须在业务查询之前。放在 pkg/db.Init 里，建连成功后立刻调用。
// 重复调用无害（gorm 的 Register 同名回调是替换语义）。
// Install 给指定的 *gorm.DB 装上统计回调。
//
// 时机：必须在业务查询之前。放在 pkg/db.Init 里，建连成功后立刻调用。
// 重复调用无害（gorm 的 Register 同名回调是替换语义）。
//
// 锚点是 gorm 各条回调链上**内置执行回调**的名字，Before 挂在执行前、
// After 挂在执行后，才能把真正的执行包住。名字写错时 gorm 只打一行日志
// 不会报错，所以这里显式写成两组，便于逐个核对。
// Install 给指定的 *gorm.DB 装上统计回调。
//
// 时机：必须在业务查询之前。放在 pkg/db.Init 里，建连成功后立刻调用。
// 重复调用无害（gorm 的 Register 同名回调是替换语义）。
//
// 锚点是 gorm 各条回调链上**内置执行回调**的名字，Before 挂在执行前、
// After 挂在执行后，才能把真正的执行包住。名字写错时 gorm 只打一行日志
// 不会报错，所以这里逐个显式写死，便于核对、也避免再引一个回调类型。
func Install(db *gorm.DB) {
	if db == nil {
		return
	}
	db.Callback().Create().Before("gorm:create").Register("sqlstats:before", before)
	db.Callback().Create().After("gorm:create").Register("sqlstats:after", after)
	db.Callback().Query().Before("gorm:query").Register("sqlstats:before", before)
	db.Callback().Query().After("gorm:query").Register("sqlstats:after", after)
	db.Callback().Update().Before("gorm:update").Register("sqlstats:before", before)
	db.Callback().Update().After("gorm:update").Register("sqlstats:after", after)
	db.Callback().Delete().Before("gorm:delete").Register("sqlstats:before", before)
	db.Callback().Delete().After("gorm:delete").Register("sqlstats:after", after)
	db.Callback().Row().Before("gorm:row").Register("sqlstats:before", before)
	db.Callback().Row().After("gorm:row").Register("sqlstats:after", after)
	db.Callback().Raw().Before("gorm:raw").Register("sqlstats:before", before)
	db.Callback().Raw().After("gorm:raw").Register("sqlstats:after", after)

	mu.Lock()
	on = true
	mu.Unlock()
}

const ctxKey = "sqlstats:start"

func before(tx *gorm.DB) {
	if tx == nil || tx.Statement == nil {
		return
	}
	tx.Statement.Set(ctxKey, time.Now())
}

func after(tx *gorm.DB) {
	if tx == nil || tx.Statement == nil {
		return
	}
	raw, ok := tx.Statement.Get(ctxKey)
	if !ok {
		return
	}
	start, ok := raw.(time.Time)
	if !ok {
		return
	}
	took := time.Since(start)
	sql := strings.TrimSpace(tx.Statement.SQL.String())
	if sql == "" {
		return
	}
	key := normalize(sql)

	mu.Lock()
	defer mu.Unlock()
	if !on {
		return
	}
	e, ok := stats[key]
	if !ok {
		// 满了先淘汰最久没执行的，避免无上限增长
		if len(stats) >= MaxEntries {
			evict()
		}
		e = &entry{}
		e.SQL = key
		e.Min = took
		stats[key] = e
	}
	e.Count++
	e.Total += took
	e.LastTook = took
	e.LastAt = time.Now()
	e.Rows += tx.Statement.RowsAffected
	if tx.Statement.Error != nil {
		e.Errors++
	}
	if took > e.Max {
		e.Max = took
	}
	if took < e.Min || e.Min == 0 {
		e.Min = took
	}
}

// evict 淘汰最久没执行的一条。调用方必须持有写锁。
func evict() {
	var oldestKey string
	var oldestAt time.Time
	first := true
	for k, e := range stats {
		if first || e.LastAt.Before(oldestAt) {
			oldestKey, oldestAt, first = k, e.LastAt, false
		}
	}
	if oldestKey != "" {
		delete(stats, oldestKey)
	}
}

// Snapshot 取当前全部统计，按累计耗时降序（最值得关注的最靠前）。
func Snapshot() []Stat {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Stat, 0, len(stats))
	for _, e := range stats {
		out = append(out, e.Stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Count > out[j].Count
	})
	return out
}

// Total 累计执行次数，给页面顶部用。
func Total() (count, errors int64) {
	mu.RLock()
	defer mu.RUnlock()
	for _, e := range stats {
		count += e.Count
		errors += e.Errors
	}
	return
}

// Reset 清空统计（页面上的"清空"按钮用）。
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	stats = map[string]*entry{}
}

// ---------------------------------------------------------------- SQL 归一化

var (
	reString = regexp.MustCompile(`'(?:[^']|'')*'`)
	reNumber = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
	reIn     = regexp.MustCompile(`(?i)\bIN\s*\(\s*\?(?:\s*,\s*\?)*\s*\)`)
	reSpace  = regexp.MustCompile(`\s+`)
)

// normalize 把一条具体 SQL 归成模板：字符串和数字字面量换成 ?，
// IN (?, ?, ?) 折叠成 IN (?)，空白压缩。
//
// 不做这步的话，同一句 `where user_id = 1` 和 `= 2` 会被当成两条，
// 统计立刻失去意义 —— druid 也是这么归的。
func normalize(sql string) string {
	s := reString.ReplaceAllString(sql, "?")
	s = reNumber.ReplaceAllString(s, "?")
	s = reIn.ReplaceAllString(s, "IN (?)")
	s = reSpace.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	const limit = 1000
	if len(s) > limit {
		s = s[:limit] + " ..."
	}
	return s
}
