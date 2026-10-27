package page

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newCtx(query string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test?"+query, nil)
	return c
}

func TestFromRequestDefaults(t *testing.T) {
	p := FromRequest(newCtx(""))
	assert.Equal(t, 1, p.PageNum, "pageNum 缺省 1（Convert.toInt(null,1)）")
	assert.Equal(t, 10, p.PageSize, "pageSize 缺省 10")
	assert.Equal(t, "asc", p.IsAsc, "isAsc 缺省 asc")
	assert.True(t, p.Reasonable, "reasonable 参数缺失时 getReasonable() 兜底 TRUE")
	assert.Empty(t, p.OrderBy("u"))
}

func TestFromRequestParsing(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		pageNum    int
		pageSize   int
		orderCol   string
		isAsc      string
		reasonable bool
	}{
		{"正常入参", "pageNum=2&pageSize=20", 2, 20, "", "asc", true},
		{"非法数字回退默认", "pageNum=abc&pageSize=xyz", 1, 10, "", "asc", true},
		{"前端 ascending 归一", "orderByColumn=createTime&isAsc=ascending", 1, 10, "createTime", "ascending", true},
		{"前端 descending 归一", "orderByColumn=createTime&isAsc=descending", 1, 10, "createTime", "descending", true},
		{"reasonable 显式 false", "reasonable=false", 1, 10, "", "asc", false},
		{"reasonable 显式 true", "reasonable=true", 1, 10, "", "asc", true},
		{"空 isAsc 不改默认", "isAsc=", 1, 10, "", "asc", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := FromRequest(newCtx(tc.query))
			assert.Equal(t, tc.pageNum, p.PageNum)
			assert.Equal(t, tc.pageSize, p.PageSize)
			assert.Equal(t, tc.orderCol, p.OrderByColumn)
			assert.Equal(t, tc.isAsc, p.IsAsc)
			assert.Equal(t, tc.reasonable, p.Reasonable)
		})
	}
}

func TestFromRequestFormOnly(t *testing.T) {
	// 契约：只从 query/form 取，不读 JSON body。
	// 这里模拟 POST form 场景，确认 PostForm 也能取到。
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request.PostForm = map[string][]string{"pageNum": {"3"}, "pageSize": {"5"}}
	p := FromRequest(c)
	assert.Equal(t, 3, p.PageNum)
	assert.Equal(t, 5, p.PageSize)
}

func TestOrderBy(t *testing.T) {
	cases := []struct {
		name  string
		p     *PageDomain
		alias string
		want  string
	}{
		{"无排序列", &PageDomain{}, "u", ""},
		{"驼峰转下划线+别名", &PageDomain{OrderByColumn: "createTime", IsAsc: "asc"}, "u", "u.create_time asc"},
		{"descending 归一", &PageDomain{OrderByColumn: "createTime", IsAsc: "descending"}, "u", "u.create_time desc"},
		{"非法方向按 asc", &PageDomain{OrderByColumn: "createTime", IsAsc: "1;drop"}, "u", "u.create_time asc"},
		{"已带别名不重复加", &PageDomain{OrderByColumn: "u.createTime", IsAsc: "desc"}, "u", "u.create_time desc"},
		{"无别名", &PageDomain{OrderByColumn: "nickName", IsAsc: "asc"}, "", "nick_name asc"},
		{"SQL 注入被剔除", &PageDomain{OrderByColumn: "createTime;drop table", IsAsc: "asc"}, "u", ""},
		{"注释符被剔除", &PageDomain{OrderByColumn: "create--time", IsAsc: "asc"}, "u", ""},
		{"union 关键字被剔除", &PageDomain{OrderByColumn: "union", IsAsc: "asc"}, "u", ""},
		{"非法字符被剔除", &PageDomain{OrderByColumn: "create`time", IsAsc: "asc"}, "u", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.p.OrderBy(tc.alias))
		})
	}
}

func TestToUnderScoreCase(t *testing.T) {
	// 预期值全部按 Java 的 StringUtils.toUnderScoreCase 逐字符推演得出。
	// 注意连续大写位于串尾时会多出一个下划线（"ABC" -> "ab_c"、"loginIP" -> "login_i_p"），
	// 这是原实现的既有行为，Go 侧忠实复刻、不做"修正"。
	cases := map[string]string{
		"":              "",
		"createTime":    "create_time",
		"userName":      "user_name",
		"HTTPServer":    "http_server",
		"userId":        "user_id",
		"id":            "id",
		"ABC":           "ab_c",
		"loginIP":       "login_i_p",
		"already_snake": "already_snake",
	}
	for in, want := range cases {
		assert.Equal(t, want, toUnderScoreCase(in), "input=%q", in)
	}
}

func TestFix(t *testing.T) {
	cases := []struct {
		name        string
		p           *PageDomain
		total       int64
		wantPageNum int
		wantOffset  int
	}{
		{"pageNum<=0 合理化", &PageDomain{PageNum: 0, PageSize: 10, Reasonable: true}, 100, 1, 0},
		{"pageNum 负数", &PageDomain{PageNum: -3, PageSize: 10, Reasonable: true}, 100, 1, 0},
		{"pageNum 越界取末页", &PageDomain{PageNum: 999, PageSize: 10, Reasonable: true}, 25, 3, 20},
		{"pageNum 正好末页", &PageDomain{PageNum: 3, PageSize: 10, Reasonable: true}, 25, 3, 20},
		{"reasonable=false 不纠偏", &PageDomain{PageNum: 999, PageSize: 10, Reasonable: false}, 25, 999, 9980},
		{"reasonable=false 仍修正负数", &PageDomain{PageNum: -1, PageSize: 10, Reasonable: false}, 25, 1, 0},
		{"空结果集", &PageDomain{PageNum: 5, PageSize: 10, Reasonable: true}, 0, 5, 40},
		{"pageSize 非法回退", &PageDomain{PageNum: 2, PageSize: 0, Reasonable: true}, 100, 2, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.p.fix(tc.total)
			assert.Equal(t, tc.wantPageNum, tc.p.PageNum)
			assert.Equal(t, tc.wantOffset, tc.p.Offset())
		})
	}
}
