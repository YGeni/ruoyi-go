// Package page 复刻 Java 侧 TableSupport + PageDomain + PageHelper 的分页契约。
//
// 契约来源（实测源码）：
//   - ruoyi-common/src/main/java/com/ruoyi/common/core/page/TableSupport.java:16-49
//     pageNum / pageSize / orderByColumn / isAsc / reasonable 五个参数，
//     **只从 query string 或 form 取**（ServletUtils.getParameter），不读 JSON body。
//   - ruoyi-common/src/main/java/com/ruoyi/common/core/page/PageDomain.java:66-99
//     默认值 pageNum=1 / pageSize=10 / isAsc="asc" / reasonable=true；
//     setIsAsc 把前端的 "ascending"/"descending" 归一为 "asc"/"desc"；
//     getOrderBy() = StringUtils.toUnderScoreCase(orderByColumn) + " " + isAsc。
//   - ruoyi-common/src/main/java/com/ruoyi/common/utils/PageUtils.java:18-26
//     orderBy 先过 SqlUtil.escapeOrderBySql，再 PageHelper.startPage(...).setReasonable(...)。
//
// 返回体侧由 pkg/result.TableDataInfo 负责（{total,rows,code:200,msg:"查询成功"}）。
package page

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 分页参数名，与 TableSupport.java:16-30 的常量字面量一致。
const (
	KeyPageNum       = "pageNum"
	KeyPageSize      = "pageSize"
	KeyOrderByColumn = "orderByColumn"
	KeyIsAsc         = "isAsc"
	KeyReasonable    = "reasonable"
)

// 默认值，与 PageDomain.java 字段初值 + Convert.toInt 的兜底一致。
const (
	DefaultPageNum    = 1
	DefaultPageSize   = 10
	DefaultIsAsc      = "asc"
	DefaultReasonable = true
)

// PageDomain 等价于 Java 的 PageDomain。
type PageDomain struct {
	PageNum       int
	PageSize      int
	OrderByColumn string
	IsAsc         string
	Reasonable    bool
}

// sqlPattern 等价于 SqlUtil.SQL_PATTERN（ruoyi-common/.../utils/sql/SqlUtil.java:20）。
//
// Java 语义是「匹配到即非法，剔除排序」，因此这里同样用「命中即拒绝」。
var sqlPattern = regexp.MustCompile(`(?:')|(?:--)|(/\*(?:.|[\n\r])*?\*/)|(\b(select|update|union|and|or|delete|insert|trancate|char|into|substr|ascii|declare|exec|master|drop|execute)\b)`)

// columnPattern 排序列白名单：只允许 标识符[.标识符]。
//
// DIFF(review): Java 侧只做了「命中黑名单就剔除」的黑名单过滤，白名单之外的内容
// （例如 a`b、func(x)）仍会被拼进 ORDER BY。Go 侧在黑名单之外**再加一道白名单**，
// 因为要拼接表别名前缀，必须确保列名是纯标识符；这是收紧而非放宽，不改变正常入参行为。
var columnPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// FromRequest 从 HTTP 请求构造 PageDomain，等价于 TableSupport.getPageDomain()。
//
// 只取 query string 与 form（c.Query / c.PostForm），**不读 JSON body**，
// 与 ServletUtils.getParameter 的语义一致。
func FromRequest(c *gin.Context) *PageDomain {
	p := &PageDomain{
		PageNum:    DefaultPageNum,
		PageSize:   DefaultPageSize,
		IsAsc:      DefaultIsAsc,
		Reasonable: DefaultReasonable,
	}
	p.PageNum = toInt(param(c, KeyPageNum), DefaultPageNum)
	p.PageSize = toInt(param(c, KeyPageSize), DefaultPageSize)
	p.OrderByColumn = strings.TrimSpace(param(c, KeyOrderByColumn))
	// setIsAsc 只在非空时赋值（PageDomain.java:83-91）
	if v := strings.TrimSpace(param(c, KeyIsAsc)); v != "" {
		p.IsAsc = v
	}
	// setReasonable(ServletUtils.getParameterToBool(REASONABLE))：
	// 参数缺失时 Convert.toBool(null) 返回 null，getReasonable() 再兜底为 TRUE
	// （Convert.java:592-595 + PageDomain.java:93-99）。
	if v := strings.TrimSpace(param(c, KeyReasonable)); v != "" {
		p.Reasonable = strings.EqualFold(v, "true") || v == "1"
	}
	return p
}

// param 取 query 或 form 的第一个非空值。
func param(c *gin.Context, key string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	return c.PostForm(key)
}

func toInt(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// OrderBy 返回 "列名 方向" 排序表达式，等价于 PageDomain.getOrderBy() +
// SqlUtil.escapeOrderBySql() 的组合结果。
//
// alias 用于给列名加表别名前缀，例如 "u" -> "u.create_time desc"。
//
// 必须传别名的原因（实测）：SysUserMapper.xml:59 的 selectUserList 是
// sys_user u LEFT JOIN sys_dept d，裸列名 create_time 在 MySQL 下直接报
// ERROR 1052 Column 'create_time' in order clause is ambiguous。
//
// DIFF(review): Java 侧把裸列名交给 PageHelper 拼接，在带 join 的列表上
// 一旦前端传了 orderByColumn 就会触发 1052；Go 侧通过 alias 显式限定，
// 规避该歧义。前端默认不传 orderByColumn，故正常路径行为一致。
func (p *PageDomain) OrderBy(alias string) string {
	if p == nil || p.OrderByColumn == "" {
		return ""
	}
	col := toUnderScoreCase(p.OrderByColumn)
	if !columnPattern.MatchString(col) || sqlPattern.MatchString(col) {
		return ""
	}
	if alias != "" && !strings.Contains(col, ".") {
		col = alias + "." + col
	}
	return col + " " + p.direction()
}

// direction 归一排序方向，等价于 PageDomain.setIsAsc 的兼容逻辑。
//
// DIFF(review): Java 对 "ascending"/"descending" 之外的任意值原样拼进 SQL
// （会产生 SQL 语法错误）；Go 侧非法值一律按 "asc" 处理，避免 500。
func (p *PageDomain) direction() string {
	switch strings.ToLower(strings.TrimSpace(p.IsAsc)) {
	case "desc", "descending":
		return "desc"
	default:
		return "asc"
	}
}

// toUnderScoreCase 等价于 StringUtils.toUnderScoreCase（驼峰转下划线）。
func toUnderScoreCase(s string) string {
	if s == "" {
		return ""
	}
	var sb strings.Builder
	runes := []rune(s)
	for i, c := range runes {
		preIsUpper := false
		if i > 0 {
			preIsUpper = isUpper(runes[i-1])
		}
		curIsUpper := isUpper(c)
		nextIsUpper := false
		if i < len(runes)-1 {
			nextIsUpper = isUpper(runes[i+1])
		}
		if preIsUpper && curIsUpper && !nextIsUpper {
			sb.WriteRune('_')
		} else if i != 0 && !preIsUpper && curIsUpper {
			sb.WriteRune('_')
		}
		sb.WriteRune(toLower(c))
	}
	return sb.String()
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// fix 分页参数合理化，等价于 PageHelper 的 reasonable=true：
//   - pageNum <= 0 -> 1
//   - pageNum > 总页数 -> 最后一页
//
// reasonable=false 时仅保证 pageNum >= 1（PageHelper 同样不会让 pageNum 为负）。
func (p *PageDomain) fix(total int64) {
	if p == nil {
		return
	}
	if p.PageNum < 1 {
		p.PageNum = DefaultPageNum
	}
	if p.PageSize < 1 {
		p.PageSize = DefaultPageSize
	}
	if !p.Reasonable || total <= 0 {
		return
	}
	pages := int((total + int64(p.PageSize) - 1) / int64(p.PageSize))
	if pages > 0 && p.PageNum > pages {
		p.PageNum = pages
	}
}

// Offset 返回当前页起始行号。必须先经过 fix(total)。
func (p *PageDomain) Offset() int {
	if p == nil || p.PageNum < 1 {
		return 0
	}
	return (p.PageNum - 1) * p.PageSize
}

// Paginate 执行「先 count 再查当页」的分页查询，返回总条数。
//
// alias 为排序列表别名（如 "u"），无 join 时传空串。
// countExpr 为空时用 COUNT(*)，否则用 COUNT(<countExpr>)，
// 例如 select distinct 的列表传 "DISTINCT u.user_id"
// （对应 PageHelper 把原 SQL 包成 select count(0) from (...) 的行为）。
//
// 注意：db 上的 Where/Joins/Scopes 等条件会被完整保留；
// 调用方传进来的 db 不应自带 Order/Limit/Offset。
// NoLimitPageSize 给"Java 侧没有 startPage() 的列表接口"用的页大小。
const NoLimitPageSize = 10000

// NoLimit 返回一个不分页的 PageDomain。
//
// 用于对齐 Java 侧**没写 startPage()**、直接 success(list) 的端点，
// 例如 SysDeptController.java:43-47 的 /system/dept/list。
//
// 注意**不能**直接传 &PageDomain{}：Paginate 末行是无条件的
// q.Limit(p.PageSize)，PageSize=0 会拼出 LIMIT 0 -> 返回空列表。
func NoLimit() *PageDomain {
	return &PageDomain{PageNum: 1, PageSize: NoLimitPageSize, Reasonable: DefaultReasonable}
}

func Paginate[T any](db *gorm.DB, p *PageDomain, alias, countExpr string, dest *[]T) (int64, error) {
	if p == nil {
		p = &PageDomain{PageNum: DefaultPageNum, PageSize: DefaultPageSize, Reasonable: DefaultReasonable}
	}
	var total int64
	// **必须强制 Statement 克隆**：gorm 的 Session（chainable_api.go / gorm.go）
	// 只在传了 Context / PrepareStmt / SkipHooks 时才 clone Statement，
	// 否则 countDB 与 db **共用同一个 Statement**。
	// 下面的 Selects=nil 会直接改到主查询上，把 "DISTINCT r.*" 一起清掉 ——
	// 实测 /system/role/list 因此返回了重复行（role_id=2 出现两次，而 COUNT 是 2 条）。
	// 传一个等价的 Context 是 gorm 官方支持的"强制 clone"写法。
	countDB := db.Session(&gorm.Session{Context: db.Statement.Context})
	// 必须清掉主查询的 Select：GORM 的 Count 会把 Selects[0] 当成 count(<expr>) 的参数，
	// 遇到 "u.*" 会拼出 count(`u`.`*`)，MySQL 报 1054 Unknown column。
	countDB.Statement.Selects = nil
	delete(countDB.Statement.Clauses, "SELECT")
	if strings.TrimSpace(countExpr) == "" {
		if err := countDB.Count(&total).Error; err != nil {
			return 0, fmt.Errorf("统计总数失败: %w", err)
		}
	} else {
		if err := countDB.Select(fmt.Sprintf("COUNT(%s)", countExpr)).Scan(&total).Error; err != nil {
			return 0, fmt.Errorf("统计总数失败: %w", err)
		}
	}
	p.fix(total)

	q := db.Session(&gorm.Session{})
	if order := p.OrderBy(alias); order != "" {
		q = q.Order(order)
	}
	if err := q.Limit(p.PageSize).Offset(p.Offset()).Find(dest).Error; err != nil {
		return 0, fmt.Errorf("分页查询失败: %w", err)
	}
	return total, nil
}
