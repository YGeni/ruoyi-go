package service

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/gentpl"
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
)

// ---------- 代码生成：对应 GenTableServiceImpl.java ----------

// 常量照搬 GenConstants.java 与 GenUtils 的默认约定。
const (
	TPLCrud        = "crud" // GenConstants.java:11
	TplWebTypeVue  = "vue2" // 前端未使用（vue 模板未移植），仅保留字段
	QueryEQ        = "EQ"
	QueryLike      = "LIKE"
	HTMLInput      = "input"
	HTMLTextarea   = "textarea"
	HTMLSelect     = "select"
	HTMLRadio      = "radio"
	HTMLCheckbox   = "checkbox"
	HTMLDatetime   = "datetime"
	HTMLImage      = "imageUpload"
	HTMLFile       = "fileUpload"
	HTMLEditor     = "editor"
	TypeString     = "String"
	TypeDate       = "Date"
	TypeLong       = "Long"
	TypeInteger    = "Integer"
	TypeDouble     = "Double"
	TypeBigDecimal = "BigDecimal"
)

// 不参与编辑/列表/查询的列名（GenConstants.java:51-60）。
var (
	columnNameNotEdit  = []string{"id", "create_by", "create_time", "del_flag"}
	columnNameNotList  = []string{"id", "create_by", "create_time", "del_flag", "update_by", "update_time"}
	columnNameNotQuery = []string{"id", "create_by", "create_time", "del_flag", "update_by", "update_time", "remark"}
	// baseEntity 里已存在的字段，domain 模板不再重复生成（GenConstants.java:62）。
	baseEntityFields = []string{"createBy", "createTime", "updateBy", "updateTime", "remark"}
	// 需要 import 的 Java 类型 -> 包名
	javaTypeImports = map[string]string{
		TypeDate:       "java.util.Date",
		TypeBigDecimal: "java.math.BigDecimal",
	}
)

// SelectGenTableList 分页查询业务表。
func SelectGenTableList(q *query.GenTableList, p *page.PageDomain) ([]entity.GenTable, int64, error) {
	return repository.SelectGenTableList(q, p)
}

// SelectDbTableList 分页查询数据库里未导入的表。
func SelectDbTableList(q *query.GenTableList, p *page.PageDomain) ([]repository.DbTable, int64, error) {
	return repository.SelectDbTableList(q, p)
}

// SelectGenTableByID 详情（含列）。
func SelectGenTableByID(tableID int64) (*entity.GenTable, error) {
	return repository.SelectGenTableByID(tableID)
}

// GenTableInfo 详情接口的返回体，对应 SysGenController.getInfo(L72-81)：
//
//	Map<String,Object> map = new HashMap<>();
//	map.put("info", table);
//	map.put("rows", columns);
//
// 前端 src/views/tool/gen/editTable.vue 读的就是 info / rows 两个键。
type GenTableInfo struct {
	Info *entity.GenTable        `json:"info"`
	Rows []entity.GenTableColumn `json:"rows"`
}

// GenTableDetail 详情。
func GenTableDetail(tableID int64) (*GenTableInfo, error) {
	t, err := repository.SelectGenTableByID(tableID)
	if err != nil {
		return nil, err
	}
	cols := t.Columns
	if cols == nil {
		cols = []entity.GenTableColumn{}
	}
	return &GenTableInfo{Info: t, Rows: cols}, nil
}

// UpdateGenTable 修改业务表 + 批量更新列。
func UpdateGenTable(t *entity.GenTable) error {
	if err := repository.UpdateGenTable(t); err != nil {
		return err
	}
	for i := range t.Columns {
		t.Columns[i].TableID = t.TableID
	}
	return repository.UpdateGenTableColumns(t.Columns)
}

// DeleteGenTableByIDs 批量删除。
func DeleteGenTableByIDs(ids []int64) error {
	return repository.DeleteGenTableByIDs(ids)
}

// ImportTable 导入表结构，等价 GenTableServiceImpl.importGenTable(L 附近)。
//
// 流程：查 information_schema.tables 拿表信息 -> 生成 gen_table 记录
// -> 查 information_schema.columns 生成列 -> 落库。
func ImportTable(tableNames []string, operator string) error {
	if len(tableNames) == 0 {
		return errors.New("请选择要导入的表")
	}
	tables, err := repository.SelectDbTableByNames(tableNames)
	if err != nil {
		return err
	}
	for _, dt := range tables {
		if err := importOne(dt, operator); err != nil {
			return err
		}
	}
	return nil
}

func importOne(dt repository.DbTable, operator string) error {
	now := time.Now()
	t := &entity.GenTable{
		TableName:      dt.TableName,
		TableComment:   dt.TableComment,
		ClassName:      convertClassName(dt.TableName),
		TplCategory:    TPLCrud,
		TplWebType:     TplWebTypeVue,
		PackageName:    "com.ruoyi.system",
		ModuleName:     moduleName(dt.TableName),
		BusinessName:   businessName(dt.TableName),
		FunctionName:   functionName(dt.TableComment),
		FunctionAuthor: "ruoyi",
		GenType:        "0",
		GenPath:        "/",
		BaseModel:      model.BaseModel{CreateBy: operator, CreateTime: &now},
	}
	one := 1
	t.FormColNum = &one
	if err := repository.InsertGenTable(t); err != nil {
		return err
	}
	cols, err := buildColumns(t.TableID, dt.TableName, operator)
	if err != nil {
		return err
	}
	return repository.ReplaceGenTableColumns(t.TableID, cols)
}

// SynchDb 同步数据库，等价 GenTableServiceImpl.synchDb(L 附近)：
// 重新读 information_schema.columns，但**保留**已存在的列配置。
func SynchDb(tableName, operator string) error {
	t, err := repository.SelectGenTableByName(tableName)
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New("表不存在")
	}
	dbCols, err := repository.SelectDbColumns(tableName)
	if err != nil {
		return err
	}
	// 旧配置按列名索引
	old := map[string]entity.GenTableColumn{}
	for _, c := range t.Columns {
		old[c.ColumnName] = c
	}
	cols := make([]entity.GenTableColumn, 0, len(dbCols))
	for i, dc := range dbCols {
		c := buildColumn(t.TableID, dc, i+1, operator)
		if o, ok := old[dc.ColumnName]; ok {
			// 保留用户改过的配置，只更新结构相关字段
			c.ColumnID = o.ColumnID
			c.IsInsert = o.IsInsert
			c.IsEdit = o.IsEdit
			c.IsList = o.IsList
			c.IsQuery = o.IsQuery
			c.QueryType = o.QueryType
			c.HTMLType = o.HTMLType
			c.DictType = o.DictType
			c.Sort = o.Sort
		}
		cols = append(cols, c)
	}
	return repository.ReplaceGenTableColumns(t.TableID, cols)
}

// CreateTable 执行建表语句，等价 GenTableServiceImpl.createTable。
func CreateTable(sql string) error {
	return repository.ExecSQL(sql)
}

func buildColumns(tableID int64, tableName, operator string) ([]entity.GenTableColumn, error) {
	dbCols, err := repository.SelectDbColumns(tableName)
	if err != nil {
		return nil, err
	}
	out := make([]entity.GenTableColumn, 0, len(dbCols))
	for i, dc := range dbCols {
		out = append(out, buildColumn(tableID, dc, i+1, operator))
	}
	return out, nil
}

// buildColumn 从 information_schema 的一列生成 gen_table_column，
// 等价 GenUtils.initColumnField（GenUtils.java:139-215）。
func buildColumn(tableID int64, dc repository.DbColumn, sort int, operator string) entity.GenTableColumn {
	now := time.Now()
	c := entity.GenTableColumn{
		TableID:       tableID,
		ColumnName:    dc.ColumnName,
		ColumnComment: dc.ColumnComment,
		ColumnType:    dc.ColumnType,
		JavaField:     toCamelCase(dc.ColumnName),
		JavaType:      TypeString,
		QueryType:     QueryEQ,
		IsPk:          b2s(strings.EqualFold(dc.ColumnKey, "PRI")),
		IsIncrement:   b2s(strings.Contains(strings.ToLower(dc.Extra), "auto_increment")),
		CreateBy:      operator,
		CreateTime:    &now,
	}
	sortCopy := sort
	c.Sort = &sortCopy
	dbType := dbTypeOf(dc.ColumnType)
	c.JavaType = javaTypeOf(dbType, dc.ColumnType)
	// 必填：非主键且数据库里 not null
	c.IsRequired = b2s(c.IsPk != "1" && strings.EqualFold(dc.IsNullable, "NO"))
	c.IsInsert = b2s(c.IsPk != "1")
	lower := strings.ToLower(dc.ColumnName)
	c.IsEdit = b2s(c.IsPk != "1" && !contains(columnNameNotEdit, lower))
	c.IsList = b2s(c.IsPk != "1" && !contains(columnNameNotList, lower))
	c.IsQuery = b2s(c.IsPk != "1" && !contains(columnNameNotQuery, lower))
	// 查询方式：name 结尾用模糊
	if strings.HasSuffix(lower, "name") {
		c.QueryType = QueryLike
	}
	c.HTMLType = htmlTypeOf(lower, dbType)
	return c
}

// ---------- 类型映射 ----------

// dbTypeOf 把 `varchar(64)` 这类带长度的类型归一成基础类型名。
func dbTypeOf(columnType string) string {
	t := strings.ToLower(columnType)
	if i := strings.Index(t, "("); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

// javaTypeOf 数据库类型 -> Java 类型，等价 GenUtils 里的 javaType 推断。
func javaTypeOf(dbType, rawType string) string {
	switch dbType {
	case "char", "varchar", "nvarchar", "varchar2", "tinytext", "text", "mediumtext", "longtext", "json":
		return TypeString
	case "datetime", "time", "date", "timestamp", "year":
		return TypeDate
	case "tinyint", "smallint", "mediumint":
		return TypeInteger
	case "int", "integer", "number", "bigint":
		// RuoYi 里 number/bigint 都归 Long（GenUtils 对 bigint 用 Long）
		if dbType == "bigint" || dbType == "number" {
			return TypeLong
		}
		return TypeInteger
	case "float", "double":
		return TypeDouble
	case "decimal", "numeric":
		return TypeBigDecimal
	case "bit", "boolean", "bool":
		return "Boolean"
	}
	// 带长度的类型（如 int(11) unsigned）回落到 raw 再判一次
	if strings.Contains(rawType, "unsigned") {
		return TypeLong
	}
	return TypeString
}

// htmlTypeOf 由列名推断前端控件类型（GenUtils 的后半段）。
func htmlTypeOf(lowerName, dbType string) string {
	switch {
	case strings.HasSuffix(lowerName, "image"):
		return HTMLImage
	case strings.HasSuffix(lowerName, "file"):
		return HTMLFile
	case strings.HasSuffix(lowerName, "content"):
		return HTMLEditor
	case strings.HasSuffix(lowerName, "status"):
		return HTMLSelect
	case strings.HasSuffix(lowerName, "type") || strings.HasSuffix(lowerName, "sex"):
		return HTMLRadio
	case dbType == "datetime" || dbType == "date" || dbType == "timestamp" || dbType == "time":
		return HTMLDatetime
	case dbType == "text" || dbType == "longtext" || dbType == "mediumtext":
		return HTMLTextarea
	}
	return HTMLInput
}

// ---------- 名称转换 ----------

// convertClassName 表名 -> 类名：去掉 sys_ 之类前缀后大驼峰。
// 等价 GenUtils.convertClassName（GenUtils.java:57-80）。
func convertClassName(tableName string) string {
	// 去掉 "sys_" / "gen_" 之类的前缀（RuoYi 只处理第一个下划线前的短前缀）
	if i := strings.Index(tableName, "_"); i > 0 && i <= 5 {
		tableName = tableName[i+1:]
	}
	return upperCamel(tableName)
}

func moduleName(tableName string) string {
	if i := strings.Index(tableName, "_"); i > 0 {
		return strings.ToLower(tableName[:i])
	}
	return "system"
}

func businessName(tableName string) string {
	if i := strings.Index(tableName, "_"); i > 0 {
		return strings.ToLower(strings.ReplaceAll(tableName[i+1:], "_", ""))
	}
	return strings.ToLower(tableName)
}

// functionName 表注释 -> 功能名：去掉末尾的"表"字。
func functionName(comment string) string {
	c := strings.TrimSpace(comment)
	c = strings.TrimSuffix(c, "表")
	c = strings.TrimSuffix(c, "信息")
	if c == "" {
		c = "未命名"
	}
	return c
}

// toCamelCase user_id -> userId，等价 StringUtils.toCamelCase。
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	out := parts[0]
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}

func upperCamel(s string) string {
	out := ""
	for _, p := range strings.Split(s, "_") {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}

// capJavaField 属性名首字母大写；
// Velocity 模板里对 xId 这种"第二字母已大写"的字段不改写，Go 侧同样处理。
func capJavaField(s string) string {
	if s == "" {
		return s
	}
	if len(s) > 1 && s[1] >= 'A' && s[1] <= 'Z' {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func b2s(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ---------- 渲染 ----------

// tplData 模板上下文，字段命名对齐 Velocity 模板里的 ${...} 变量。
type tplData struct {
	TableName        string
	TableComment     string
	ClassName        string
	ClassNameLower   string
	PackageName      string
	ModuleName       string
	BusinessName     string
	FunctionName     string
	FunctionAuthor   string
	Datetime         string
	PermissionPrefix string
	ParentMenuId     string
	PackageNamePath  string
	ImportList       []string
	Pk               *tplColumn
	Columns          []tplColumn
	QueryColumns     []tplColumn
	InsertColumns    []tplColumn
	UpdateColumns    []tplColumn
	ColumnNameList   string
	HasDateColumn    bool
	HasCreateTime    bool
	HasUpdateTime    bool
}

type tplColumn struct {
	ColumnName       string
	ColumnComment    string
	ExcelComment     string
	ColumnType       string
	JavaType         string
	JavaField        string
	CapJavaField     string
	IsPk             bool
	IsIncrementBool  bool
	IsRequired       bool
	IsInsert         bool
	IsEdit           bool
	IsList           bool
	IsQuery          bool
	QueryType        string
	HTMLType         string
	DictType         string
	IsSuperColumn    bool
	IsDateType       bool
	IsStringType     bool
	IsRequiredString bool
	SQLOp            string
	SQLRight         string
}

// buildTplData 组装模板上下文。
func buildTplData(t *entity.GenTable) *tplData {
	d := &tplData{
		TableName:        t.TableName,
		TableComment:     t.TableComment,
		ClassName:        t.ClassName,
		ClassNameLower:   lowerFirst(t.ClassName),
		PackageName:      t.PackageName,
		ModuleName:       t.ModuleName,
		BusinessName:     t.BusinessName,
		FunctionName:     t.FunctionName,
		FunctionAuthor:   t.FunctionAuthor,
		Datetime:         time.Now().Format("2006-01-02"),
		PermissionPrefix: fmt.Sprintf("%s:%s", t.ModuleName, t.BusinessName),
		ParentMenuId:     "3",
		PackageNamePath:  strings.ReplaceAll(t.PackageName, ".", "/"),
	}
	importSet := map[string]bool{}
	var names []string
	for _, c := range t.Columns {
		tc := tplColumn{
			ColumnName:       c.ColumnName,
			ColumnComment:    c.ColumnComment,
			ExcelComment:     excelComment(c.ColumnComment),
			ColumnType:       c.ColumnType,
			JavaType:         c.JavaType,
			JavaField:        c.JavaField,
			CapJavaField:     capJavaField(c.JavaField),
			IsPk:             c.IsPk == "1",
			IsIncrementBool:  c.IsIncrement == "1",
			IsRequired:       c.IsRequired == "1",
			IsInsert:         c.IsInsert == "1",
			IsEdit:           c.IsEdit == "1",
			IsList:           c.IsList == "1",
			IsQuery:          c.IsQuery == "1",
			QueryType:        c.QueryType,
			HTMLType:         c.HTMLType,
			DictType:         c.DictType,
			IsSuperColumn:    contains(baseEntityFields, c.JavaField),
			IsDateType:       c.JavaType == TypeDate,
			IsStringType:     c.JavaType == TypeString,
			IsRequiredString: c.JavaType == TypeString && c.IsRequired == "1",
		}
		tc.SQLOp, tc.SQLRight = sqlOp(tc)
		d.Columns = append(d.Columns, tc)
		if tc.IsQuery {
			d.QueryColumns = append(d.QueryColumns, tc)
		}
		if tc.IsInsert {
			d.InsertColumns = append(d.InsertColumns, tc)
		}
		if !tc.IsPk {
			d.UpdateColumns = append(d.UpdateColumns, tc)
		}
		if tc.IsPk && d.Pk == nil {
			pk := tc
			d.Pk = &pk
		}
		if imp, ok := javaTypeImports[c.JavaType]; ok && !tc.IsSuperColumn {
			if !importSet[imp] {
				importSet[imp] = true
				names = append(names, imp)
			}
		}
		if c.JavaType == TypeDate {
			d.HasDateColumn = true
		}
		if c.JavaField == "createTime" {
			d.HasCreateTime = true
		}
		if c.JavaField == "updateTime" {
			d.HasUpdateTime = true
		}
	}
	d.ImportList = names
	// mapper.xml 的 select 列清单
	parts := make([]string, 0, len(d.Columns))
	for _, c := range d.Columns {
		parts = append(parts, c.ColumnName)
	}
	d.ColumnNameList = strings.Join(parts, ",")
	return d
}

// excelComment 去掉注释里的括号部分（domain 模板的 parentheseIndex 逻辑）。
func excelComment(comment string) string {
	if i := strings.Index(comment, "（"); i >= 0 {
		return comment[:i]
	}
	if i := strings.Index(comment, "("); i >= 0 {
		return comment[:i]
	}
	return comment
}

// sqlOp 查询类型 -> SQL 片段，对应 mapper.xml.vm 的 #if 链。
func sqlOp(c tplColumn) (op, right string) {
	ph := "#{" + c.JavaField + "}"
	switch strings.ToUpper(c.QueryType) {
	case "NE":
		return "!=", ph
	case "GT":
		return "&gt;", ph
	case "GTE":
		return "&gt;=", ph
	case "LT":
		return "&lt;", ph
	case "LTE":
		return "&lt;=", ph
	case "LIKE":
		return "like", "concat('%', " + ph + ", '%')"
	case "BETWEEN":
		capName := capJavaField(c.JavaField)
		return "between", "#{params.begin" + capName + "} and #{params.end" + capName + "}"
	}
	return "=", ph
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// RenderAll 渲染全部模板，返回 输出路径 -> 内容。
//
// 等价 VelocityUtils.prepareContext + GenTableServiceImpl.previewCode 的
// 模板渲染部分（不落盘）。
func RenderAll(t *entity.GenTable) (map[string]string, error) {
	tpls, err := gentpl.Parse()
	if err != nil {
		return nil, err
	}
	data := buildTplData(t)
	if data.Pk == nil {
		return nil, errors.New("该表没有主键列，无法生成代码")
	}
	out := map[string]string{}
	for _, f := range gentpl.Files {
		tpl := tpls[f.Tpl]
		if tpl == nil {
			continue
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, data); err != nil {
			return nil, err
		}
		// 输出路径本身也是个模板
		pt, err := gentpl.ParsePath(f.Path)
		if err != nil {
			return nil, err
		}
		var pbuf bytes.Buffer
		if err := pt.Execute(&pbuf, data); err != nil {
			return nil, err
		}
		out[pbuf.String()] = buf.String()
	}
	return out, nil
}

// GenZip 渲染并打成 zip，等价 GenTableServiceImpl.downloadCode。
func GenZip(t *entity.GenTable) ([]byte, error) {
	files, err := RenderAll(t)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for path, content := range files {
		w, err := zw.Create(path)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DbTable 数据库里的一张物理表（转发 repository.DbTable，避免 handler 直接依赖仓储）。
type DbTable = repository.DbTable

// GenTableByName 按表名查业务表（genCode 用）。
func GenTableByName(tableName string) (*entity.GenTable, error) {
	return repository.SelectGenTableByName(tableName)
}

// GenToPath 生成到 genPath 指定的自定义路径（genType='1'）。
//
// 等价 GenTableServiceImpl.generatorCode 里 genType 为 "1" 的分支：
// 直接把文件写到服务器本地磁盘，不下发下载。
func GenToPath(t *entity.GenTable) error {
	files, err := RenderAll(t)
	if err != nil {
		return err
	}
	root := t.GenPath
	if strings.TrimSpace(root) == "" || root == "/" {
		return errors.New("生成路径未配置")
	}
	for p, content := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// GenColumns 查业务表的列，等价 GenTableColumnServiceImpl
// .selectGenTableColumnListByTableId（GenController.columnList L99-107）。
func GenColumns(tableID int64) ([]entity.GenTableColumn, error) {
	rows, err := repository.SelectGenTableColumns(tableID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []entity.GenTableColumn{}
	}
	return rows, nil
}

// GenZipBatch 多张表打成一个 zip，等价 GenTableServiceImpl.downloadCode(String[])。
func GenZipBatch(tableNames []string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range tableNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		t, err := repository.SelectGenTableByName(name)
		if err != nil {
			return nil, err
		}
		if t == nil {
			continue
		}
		files, err := RenderAll(t)
		if err != nil {
			return nil, err
		}
		for path, content := range files {
			w, err := zw.Create(path)
			if err != nil {
				return nil, err
			}
			if _, err := w.Write([]byte(content)); err != nil {
				return nil, err
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
