package router

// 本文件是 AGENTS.md §12「风险 1」的另一半：权限串绑定遗漏。
//
// 做法不是"人肉核对"，而是**在测试里直接解析 Java 控制器源码**，
// 抽出每个端点的 (method, path, @ss.hasPermi 权限串)，
// 与本包 perm_table.go 的 permRoutes 做双向 diff：
//
//	缺失：Java 有注解、Go 未登记        -> t.Fatal（CI 红）
//	多余：Go 登记了、Java 侧不存在      -> t.Fatal（CI 红）
//	二期延后：Java 有、但本轮不实现路由 -> 只登记到报告，不算失败
//
// 报告落盘 docs/go-migration/02b-perm-diff-report.md，可直接作为评审/回归凭据。

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// javaRoots 需要扫描的 Java 控制器目录（相对仓库根，由 repoRoot 定位）。
//
// 覆盖 ruoyi-admin + ruoyi-generator + ruoyi-quartz，
// 保证"多余"判定不会因为漏扫模块而误报（如 monitor:job:list 来自 quartz）。
var javaRootSuffixes = []string{
	"ruoyi-admin/src/main/java/com/ruoyi/web/controller",
	"ruoyi-generator/src/main/java/com/ruoyi/generator/controller",
	"ruoyi-quartz/src/main/java/com/ruoyi/quartz/controller",
}

// repoRoot 从 internal/router 向上找仓库根（以存在 ruoyi-admin 目录为标志）。
//
// 不写死 "../../../" 是为了让 `go test ./...`（工作目录是包目录）与
// 直接 gofmt/IDE 运行都能正确定位。
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "ruoyi-admin")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("未能定位仓库根（向上 8 层未找到 ruoyi-admin 目录）")
	return ""
}

func javaRoots(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	out := make([]string, 0, len(javaRootSuffixes))
	for _, s := range javaRootSuffixes {
		out = append(out, filepath.Join(root, s))
	}
	return out
}

// scopeFiles 本期（1b）落地的控制器；其余归入二期延后。
var scopeFiles = map[string]bool{
	"SysUserController.java":     true,
	"SysRoleController.java":     true,
	"SysDeptController.java":     true,
	"SysPostController.java":     true,
	"SysDictTypeController.java": true,
	"SysDictDataController.java": true,
	"SysConfigController.java":   true,
	"SysNoticeController.java":   true,
	"SysProfileController.java":  true,
	"CommonController.java":      true,
	// 菜单管理原属二期清单，因前端"系统管理 > 菜单管理"页面直接报错而提前落地。
	"SysMenuController.java":       true,
	"SysUserOnlineController.java": true,
}

// deferPathKeywords 本期明确不做的路径（Excel 导入导出，见 commit message 遗留项）。
var deferPathKeywords = []string{"export", "importData", "importTemplate"}

// javaEndpoint 一个 Java 控制器端点。
type javaEndpoint struct {
	File   string // 控制器文件名
	Line   int
	Method string
	Path   string // gin 风格
	Perm   string // 空串表示无 @PreAuthorize
}

var (
	reClassDecl   = regexp.MustCompile(`\bclass\s+\w+`)
	reClsMapping  = regexp.MustCompile(`@RequestMapping\(\s*"([^"]*)"`)
	reMapping     = regexp.MustCompile(`@(Get|Post|Put|Delete|Request)Mapping\((.*)\)`)
	reMappingBare = regexp.MustCompile(`^@(Get|Post|Put|Delete|Request)Mapping$`)
	rePathArgs    = regexp.MustCompile(`"([^"]*)"`)
	reReqMethod   = regexp.MustCompile(`RequestMethod\.(\w+)`)
	reHasPermi    = regexp.MustCompile(`@ss\.has(?:Permi|AnyPermi)\('([^']*)'`)
	reMethodSig   = regexp.MustCompile(`^(public|private|protected)\s`)
	reBrace       = regexp.MustCompile(`\{(\w+)\}`)
)

// parseJavaController 解析单个控制器，返回其全部端点。
//
// 解析规则对齐 Spring MVC 语义：
//   - 类级 @RequestMapping 作为路径前缀
//   - 方法级 @XxxMapping（可无参数，路径即类前缀；可用 value={"/a","/b"} 声明多路径）
//   - @PreAuthorize("@ss.hasPermi('...')") 取权限串
//   - Java 的 {id} 路径变量转成 gin 的 :id
func parseJavaController(path string) ([]javaEndpoint, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")

	clsIdx := 0
	for i, l := range lines {
		if reClassDecl.MatchString(l) {
			clsIdx = i
			break
		}
	}
	prefix := ""
	for _, l := range lines[:clsIdx] {
		if m := reClsMapping.FindStringSubmatch(l); m != nil {
			prefix = m[1]
		}
	}

	var out []javaEndpoint
	cur := struct {
		line  int
		maps  [][2]string
		perm  string
		buf   string
		start bool
	}{}

	flush := func() {
		if !cur.start {
			return
		}
		for _, mp := range cur.maps {
			full := strings.ReplaceAll(prefix+mp[1], "//", "/")
			out = append(out, javaEndpoint{
				File:   filepath.Base(path),
				Line:   cur.line,
				Method: strings.ToUpper(mp[0]),
				Path:   reBrace.ReplaceAllString(full, ":$1"),
				Perm:   cur.perm,
			})
		}
		cur.maps, cur.perm, cur.buf, cur.start = nil, "", "", false
	}

	for i := clsIdx; i < len(lines); i++ {
		s := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(s, "@"):
			if !cur.start {
				cur.start, cur.line = true, i+1
			}
			if m := reHasPermi.FindStringSubmatch(s); m != nil {
				cur.perm = m[1]
			}
			// 无括号注解（@PostMapping）：路径取类前缀
			if mm := reMappingBare.FindStringSubmatch(s); mm != nil {
				cur.maps = append(cur.maps, [2]string{mm[1], ""})
				continue
			}
			cur.buf += s
			if strings.Count(cur.buf, "(") > 0 && strings.Count(cur.buf, "(") == strings.Count(cur.buf, ")") {
				if mm := reMapping.FindStringSubmatch(cur.buf); mm != nil {
					kind, args := mm[1], mm[2]
					paths := rePathArgs.FindAllStringSubmatch(args, -1)
					if len(paths) == 0 {
						paths = [][]string{{"", ""}}
					}
					if meths := reReqMethod.FindAllStringSubmatch(args, -1); len(meths) > 0 {
						for _, me := range meths {
							for _, p := range paths {
								cur.maps = append(cur.maps, [2]string{me[1], p[1]})
							}
						}
					} else if kind != "Request" {
						for _, p := range paths {
							cur.maps = append(cur.maps, [2]string{kind, p[1]})
						}
					}
				}
				cur.buf = ""
			}
		case reMethodSig.MatchString(s) && strings.Contains(s, "(") && strings.Contains(s, ")"):
			flush()
		}
	}
	flush()
	return out, nil
}

func loadJavaEndpoints(t *testing.T) []javaEndpoint {
	t.Helper()
	var all []javaEndpoint
	for _, root := range javaRoots(t) {
		// 必须递归：ruoyi-admin 的控制器分散在 system/ common/ monitor/ tool/ 子包下
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), "Controller.java") {
				return nil
			}
			eps, perr := parseJavaController(path)
			if perr != nil {
				return perr
			}
			all = append(all, eps...)
			return nil
		})
		if err != nil {
			t.Fatalf("扫描 Java 控制器目录失败 %s: %v", root, err)
		}
	}
	if len(all) == 0 {
		t.Fatalf("未解析到任何 Java 端点，检查 javaRoots 路径是否正确")
	}
	return all
}

func key(method, path string) string { return method + " " + path }

// TestPermTableBidirectionalDiff 双向 diff 主断言。
func TestPermTableBidirectionalDiff(t *testing.T) {
	java := loadJavaEndpoints(t)

	var inScope, deferred []javaEndpoint
	for _, e := range java {
		skip := false
		for _, kw := range deferPathKeywords {
			if strings.Contains(e.Path, kw) {
				skip = true
				break
			}
		}
		if scopeFiles[e.File] && !skip {
			inScope = append(inScope, e)
		} else {
			deferred = append(deferred, e)
		}
	}

	goMap := make(map[string]RoutePerm, len(permRoutes))
	for _, r := range permRoutes {
		if _, dup := goMap[key(r.Method, r.Path)]; dup {
			t.Fatalf("perm_table.go 存在重复路由登记: %s %s", r.Method, r.Path)
		}
		goMap[key(r.Method, r.Path)] = r
	}

	javaMap := make(map[string]javaEndpoint, len(inScope))
	for _, e := range inScope {
		javaMap[key(e.Method, e.Path)] = e
	}

	var missing, extra []string
	for k, e := range javaMap {
		g, ok := goMap[k]
		if !ok {
			missing = append(missing, fmt.Sprintf("%s %s -> %s（%s:%d）",
				e.Method, e.Path, orEmpty(e.Perm), e.File, e.Line))
			continue
		}
		if g.Perm != e.Perm {
			missing = append(missing, fmt.Sprintf("%s %s 权限不一致：Java=%s Go=%s",
				e.Method, e.Path, orEmpty(e.Perm), orEmpty(g.Perm)))
		}
	}
	for k, g := range goMap {
		if _, ok := javaMap[k]; !ok {
			extra = append(extra, fmt.Sprintf("%s %s -> %s（Go 独有）", g.Method, g.Path, orEmpty(g.Perm)))
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	// 报告落盘（无论成败）
	writePermDiffReport(t, java, inScope, deferred, missing, extra)

	if len(missing) > 0 {
		t.Errorf("缺失（Java 有注解、Go 未登记）共 %d 条：\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("多余（Go 登记了、Java 侧不存在）共 %d 条：\n  %s", len(extra), strings.Join(extra, "\n  "))
	}
	if len(missing) == 0 && len(extra) == 0 {
		t.Logf("双向 diff 通过：范围内 %d 条全部对齐，二期延后 %d 条", len(inScope), len(deferred))
	}
}

// TestPermConstantsCoverJava 所有 Java 权限串都能在 perm_table 里找到同名常量。
// 防止常量表与 Java 字面量漂移（例如 Java 改成 system:user:resetPwd 而 Go 还是 resetPwd）。
func TestPermConstantsCoverJava(t *testing.T) {
	java := loadJavaEndpoints(t)
	perms := map[string]bool{}
	for _, e := range java {
		if e.Perm != "" {
			perms[e.Perm] = true
		}
	}
	consts := map[string]bool{}
	for _, r := range permRoutes {
		if r.Perm != "" {
			consts[r.Perm] = true
		}
	}
	// 已登记常量（含二期延后）也要能覆盖
	for _, v := range permConstants {
		consts[v] = true
	}
	var uncovered []string
	for p := range perms {
		if !consts[p] {
			uncovered = append(uncovered, p)
		}
	}
	sort.Strings(uncovered)
	if len(uncovered) > 0 {
		t.Fatalf("以下 Java 权限串在 perm_table.go 无对应常量：\n  %s", strings.Join(uncovered, "\n  "))
	}
}

// TestPhase2ListMatchesJava 二期清单必须与 Java 源码一致（双向）。
//
// permRoutesPhase2 是交接清单，最怕"改了 Java 忘了同步"或"二期落地后没从清单摘除"。
// 这里让测试直接拿 Java 解析结果当基准做比对，任一侧漂移都会红。
func TestPhase2ListMatchesJava(t *testing.T) {
	java := loadJavaEndpoints(t)
	javaDef := map[string]bool{}
	for _, e := range java {
		skip := false
		for _, kw := range deferPathKeywords {
			if strings.Contains(e.Path, kw) {
				skip = true
				break
			}
		}
		if scopeFiles[e.File] && !skip {
			continue // 范围内，由 permRoutes 负责
		}
		javaDef[key(e.Method, e.Path)] = true
	}
	goDef := map[string]bool{}
	for _, r := range permRoutesPhase2 {
		goDef[key(r.Method, r.Path)] = true
	}
	var missing, stale []string
	for k := range javaDef {
		if !goDef[k] {
			missing = append(missing, k)
		}
	}
	for k := range goDef {
		if !javaDef[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	if len(missing) > 0 {
		t.Errorf("Java 有但二期清单未登记（%d）：\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("二期清单里有 Java 已不存在/已翻入范围内的条目（%d）：\n  %s", len(stale), strings.Join(stale, "\n  "))
	}
	if len(missing) == 0 && len(stale) == 0 {
		t.Logf("二期清单与 Java 一致：%d 条（grep -n TODO(phase2) 可逐条追踪）", len(goDef))
	}
}

// TestPermRoutesNoDuplicatePath 同一 method+path 不允许登记两次。
func TestPermRoutesNoDuplicatePath(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range permRoutes {
		k := key(r.Method, r.Path)
		if seen[k] {
			t.Fatalf("重复登记: %s", k)
		}
		seen[k] = true
	}
}

// TestRegisteredRoutesHavePermEntry 已注册的 gin 路由必须能在 perm_table 里查到，
// 或者落在白名单（登录/验证码/静态资源等无需权限串的接口）。
//
// 这道断言防止"新加路由时忘记同步 perm_table"——步骤 4/5 批量注册时尤其容易漏。
func TestRegisteredRoutesHavePermEntry(t *testing.T) {
	// 无需权限串的白名单（1a 已交付 + 静态资源）
	whitelist := map[string]bool{
		"GET /captchaImage": true,
		"POST /login":       true,
		"POST /logout":      true,
		"GET /getInfo":      true,
		"GET /getRouters":   true,
		// 静态资源（router.go 的 r.Static，对齐 ResourcesConfig.java:33-34）。
		// gin 的 Static 会同时注册 GET 与 HEAD 两个方法，两个都要在白名单里，
		// 否则这道护栏会把 HEAD 报成"无归属"。
		"GET /profile/*filepath":  true,
		"HEAD /profile/*filepath": true,
	}
	registered := map[string]bool{}
	for _, r := range New().Routes() {
		registered[key(r.Method, r.Path)] = true
	}
	var unregistered []string
	for _, rp := range permRoutes {
		// 只校验"已注册"的路由必须能在表里查到；未注册的属步骤 4/5 待办，这里不报错
		if registered[key(rp.Method, rp.Path)] {
			delete(registered, key(rp.Method, rp.Path))
		}
	}
	for k := range registered {
		if !whitelist[k] {
			unregistered = append(unregistered, k)
		}
	}
	sort.Strings(unregistered)
	if len(unregistered) > 0 {
		t.Errorf("以下已注册路由在 perm_table.go / 白名单中均无归属：\n  %s",
			strings.Join(unregistered, "\n  "))
	}
}

func orEmpty(s string) string {
	if s == "" {
		return "(仅需登录)"
	}
	return s
}

// writePermDiffReport 落盘差异报告。
func writePermDiffReport(t *testing.T, all, inScope, deferred []javaEndpoint, missing, extra []string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("# 1b 权限绑定双向 diff 报告\n\n")
	b.WriteString("由 `internal/router/perm_diff_test.go` 自动生成于 `go test ./internal/router/`。\n")
	b.WriteString("数据源：Java 控制器源码注解（运行时解析，非人工抄写）。\n\n")
	fmt.Fprintf(&b, "| 项 | 数量 |\n|---|---|\n")
	fmt.Fprintf(&b, "| Java 端点总数（admin+generator+quartz） | %d |\n", len(all))
	fmt.Fprintf(&b, "| 1b 范围内端点 | %d |\n", len(inScope))
	fmt.Fprintf(&b, "| 二期延后端点 | %d |\n", len(deferred))
	fmt.Fprintf(&b, "| 缺失（Java→Go） | %d |\n", len(missing))
	fmt.Fprintf(&b, "| 多余（Go→Java） | %d |\n", len(extra))
	b.WriteString("\n## 缺失\n\n")
	if len(missing) == 0 {
		b.WriteString("（无）\n")
	} else {
		for _, m := range missing {
			b.WriteString("- " + m + "\n")
		}
	}
	b.WriteString("\n## 多余\n\n")
	if len(extra) == 0 {
		b.WriteString("（无）\n")
	} else {
		for _, m := range extra {
			b.WriteString("- " + m + "\n")
		}
	}
	b.WriteString("\n## 1b 范围内绑定清单\n\n")
	b.WriteString("| Method | Path | 权限串 | Java 来源 |\n|---|---|---|---|\n")
	sorted := append([]javaEndpoint(nil), inScope...)
	sort.Slice(sorted, func(i, j int) bool {
		return key(sorted[i].Method, sorted[i].Path) < key(sorted[j].Method, sorted[j].Path)
	})
	for _, e := range sorted {
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s:%d |\n", e.Method, e.Path, orEmpty(e.Perm), e.File, e.Line)
	}
	b.WriteString("\n## 二期延后（本轮不实现路由）\n\n")
	b.WriteString("| Method | Path | 权限串 | Java 来源 |\n|---|---|---|---|\n")
	sort.Slice(deferred, func(i, j int) bool {
		return key(deferred[i].Method, deferred[i].Path) < key(deferred[j].Method, deferred[j].Path)
	})
	for _, e := range deferred {
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s:%d |\n", e.Method, e.Path, orEmpty(e.Perm), e.File, e.Line)
	}

	dir := filepath.Join(repoRoot(t), "docs", "go-migration")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("创建报告目录失败: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "02b-perm-diff-report.md"), []byte(b.String()), 0o644); err != nil {
		t.Logf("写入报告失败: %v", err)
	}
}
