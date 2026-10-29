package system

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/jwtx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"github.com/gin-gonic/gin"
)

// startAt 进程启动时刻，用于页面上的"运行时长"。
var startAt = time.Now()

// DruidPage GET /druid/login.html
//
// DIFF(review) —— 这是**有意的功能替换**，不是等价移植，必须说清楚：
//
// Java 的"数据监控"是 druid 自带的 StatViewServlet（DruidConfig 里注册的
// ServletRegistrationBean），提供：数据源、SQL 监控、SQL 防火墙、Web 应用、
// URI 监控、Session 监控、Spring 监控、JSON API 一整套。
// 它的核心是 druid 在 JDBC 驱动层拦截每条 SQL 做的统计 —— Go 侧用
// database/sql + gorm，没有这一层，SQL 执行统计**无法**提供。
//
// 前端 src/views/monitor/druid/index.vue:11 iframe 的正是
// `${VITE_APP_BASE_API}/druid/login.html`，所以不打这个地址，
// 点"数据监控"菜单就一定是 404（druid 的页面由 druid jar 提供）。
//
// 这里给的是 Go 侧**能真实拿到**的对等信息：
//   - 数据源：database/sql 的连接池（DBStats）
//   - 缓存：Redis 连接池 + key 数量 + 内存占用
//   - 运行时：goroutine 数 / 堆内存 / GC 次数 / 进程启动时长
//
// 页面上明确标注"SQL 执行统计不可用"，不假装是 druid。
//
// 鉴权：只校验登录态（与 /unlockscreen 一样直接注册、不挂权限串）——
// Java 侧 /druid/* 也不是 Shiro 管的，是 druid 自己的 StatViewServlet
// 用登录页 + session 管的。这里用 JWT 登录态，比 druid 的默认口令更严。
func (MonitorHandler) DruidPage(c *gin.Context) {
	// iframe 发不出 Authorization 头，所以这里**不走** middleware.JWTAuth()，
	// 而是自己取：header 优先，query 兜底（前端把 token 拼在 URL 上）。
	token := jwtx.StripBearer(c.GetHeader(config.C.Token.Header))
	if token == "" {
		token = c.Query("token")
	}
	if !middleware.AuthWithToken(c, token) {
		return
	}
	data := buildDruidData()
	var buf bytes.Buffer
	if err := druidTpl.Execute(&buf, data); err != nil {
		fail(c, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// druidKV 一行指标。
type druidKV struct {
	K string
	V string
}

type druidData struct {
	Title    string
	Updated  string
	Uptime   string
	Sections []druidSection
	Note     string
}

type druidSection struct {
	Name string
	Rows []druidKV
}

func buildDruidData() druidData {
	d := druidData{
		Title:   "数据监控",
		Updated: time.Now().Format("2006-01-02 15:04:05"),
		Uptime:  time.Since(startAt).Truncate(time.Second).String(),
		Note:    "本页面由 Go 后端提供，非 druid。druid 的 SQL 执行统计依赖 JDBC 驱动层拦截，Go 侧（database/sql + gorm）没有这一层，无法提供。",
	}
	d.Sections = append(d.Sections, dataSourceSection(), redisSection(), runtimeSection())
	return d
}

// dataSourceSection 数据库 + 连接池，对应 druid 的"数据源"页。
func dataSourceSection() druidSection {
	s := druidSection{Name: "数据源"}
	sqlDB, err := db.DB.DB()
	if err != nil {
		s.Rows = append(s.Rows, druidKV{"状态", "不可用：" + err.Error()})
		return s
	}
	var dbName, dbVersion string
	db.DB.Raw("SELECT DATABASE()").Row().Scan(&dbName)
	db.DB.Raw("SELECT VERSION()").Row().Scan(&dbVersion)
	s.Rows = append(s.Rows,
		druidKV{"当前库", dbName},
		druidKV{"版本", dbVersion},
	)
	st := sqlDB.Stats()
	s.Rows = append(s.Rows,
		druidKV{"最大连接数", fmt.Sprint(st.MaxOpenConnections)},
		druidKV{"已打开连接", fmt.Sprint(st.OpenConnections)},
		druidKV{"使用中", fmt.Sprint(st.InUse)},
		druidKV{"空闲", fmt.Sprint(st.Idle)},
		druidKV{"累计等待次数", fmt.Sprint(st.WaitCount)},
		druidKV{"累计等待时长", st.WaitDuration.Truncate(time.Millisecond).String()},
		druidKV{"因超最大空闲关闭", fmt.Sprint(st.MaxIdleClosed)},
		druidKV{"因超最大生存期关闭", fmt.Sprint(st.MaxLifetimeClosed)},
	)
	return s
}

// redisSection 缓存，对应 druid 的"Session 监控"之外的缓存侧信息；
// 真实数据源是 /monitor/cache 那套（Redis INFO）。
func redisSection() druidSection {
	s := druidSection{Name: "缓存（Redis）"}
	rc := redisx.Client()
	if rc == nil {
		s.Rows = append(s.Rows, druidKV{"状态", "未初始化"})
		return s
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if pong, err := rc.Ping(ctx).Result(); err != nil {
		s.Rows = append(s.Rows, druidKV{"状态", "连接失败：" + err.Error()})
		return s
	} else {
		s.Rows = append(s.Rows, druidKV{"状态", pong})
	}
	if ps := rc.PoolStats(); ps != nil {
		s.Rows = append(s.Rows,
			druidKV{"命中次数", fmt.Sprint(ps.Hits)},
			druidKV{"未命中次数", fmt.Sprint(ps.Misses)},
			druidKV{"超时次数", fmt.Sprint(ps.Timeouts)},
			druidKV{"总连接数", fmt.Sprint(ps.TotalConns)},
			druidKV{"空闲连接数", fmt.Sprint(ps.IdleConns)},
			druidKV{"陈旧连接数", fmt.Sprint(ps.StaleConns)},
		)
	}
	if size, err := rc.DBSize(ctx).Result(); err == nil {
		s.Rows = append(s.Rows, druidKV{"key 数量", fmt.Sprint(size)})
	}
	return s
}

// runtimeSection Go 运行时，对应 druid 的"Web 应用/JVM"那类信息。
func runtimeSection() druidSection {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return druidSection{Name: "运行时", Rows: []druidKV{
		{"Go 版本", runtime.Version()},
		{"CPU 核数", fmt.Sprint(runtime.NumCPU())},
		{"Goroutine 数", fmt.Sprint(runtime.NumGoroutine())},
		{"堆已分配", formatBytes(ms.HeapAlloc)},
		{"堆系统申请", formatBytes(ms.HeapSys)},
		{"GC 次数", fmt.Sprint(ms.NumGC)},
		{"GC 累计暂停", time.Duration(ms.PauseTotalNs).Truncate(time.Millisecond).String()},
		{"进程启动时长", time.Since(startAt).Truncate(time.Second).String()},
	}}
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
