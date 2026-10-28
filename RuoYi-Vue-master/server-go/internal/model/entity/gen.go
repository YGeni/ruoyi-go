package entity

import (
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model"
)

// GenTable 代码生成业务表 gen_table。
//
// 字段与 ruoyi-generator/.../domain/GenTable.java 对齐。
// 该表有 create_time / update_time，正常嵌入 BaseModel。
type GenTable struct {
	model.BaseModel
	TableID        int64  `gorm:"column:table_id;primaryKey;autoIncrement" json:"tableId"`
	TableName      string `gorm:"column:table_name" json:"tableName"`
	TableComment   string `gorm:"column:table_comment" json:"tableComment"`
	SubTableName   string `gorm:"column:sub_table_name" json:"subTableName"`
	SubTableFkName string `gorm:"column:sub_table_fk_name" json:"subTableFkName"`
	ClassName      string `gorm:"column:class_name" json:"className"`
	TplCategory    string `gorm:"column:tpl_category" json:"tplCategory"`
	TplWebType     string `gorm:"column:tpl_web_type" json:"tplWebType"`
	PackageName    string `gorm:"column:package_name" json:"packageName"`
	ModuleName     string `gorm:"column:module_name" json:"moduleName"`
	BusinessName   string `gorm:"column:business_name" json:"businessName"`
	FunctionName   string `gorm:"column:function_name" json:"functionName"`
	FunctionAuthor string `gorm:"column:function_author" json:"functionAuthor"`
	FormColNum     *int   `gorm:"column:form_col_num" json:"formColNum"`
	GenType        string `gorm:"column:gen_type" json:"genType"`
	GenPath        string `gorm:"column:gen_path" json:"genPath"`
	Options        string `gorm:"column:options" json:"options"`

	// Columns 非持久化：子表列（gen_table_column 按 table_id 关联）。
	Columns []GenTableColumn `gorm:"-" json:"columns"`
	// PkColumn 非持久化：主键列，模板里 ${pkColumn.javaField} 会用到。
	PkColumn *GenTableColumn `gorm:"-" json:"pkColumn"`
}

// GenTableColumn 代码生成业务表字段 gen_table_column。
type GenTableColumn struct {
	// 注意：**不能**嵌 model.BaseModel —— gen_table_column 只有
	// create_by/create_time/update_by/update_time，没有 remark / searchValue /
	// params，嵌了会在插入时报
	// "Unknown column 'remark' in 'field list'"（踩过一次）。
	CreateBy   string     `gorm:"column:create_by" json:"createBy"`
	CreateTime *time.Time `gorm:"column:create_time" json:"createTime"`
	UpdateBy   string     `gorm:"column:update_by" json:"updateBy"`
	UpdateTime *time.Time `gorm:"column:update_time" json:"updateTime"`

	ColumnID      int64  `gorm:"column:column_id;primaryKey;autoIncrement" json:"columnId"`
	TableID       int64  `gorm:"column:table_id" json:"tableId"`
	ColumnName    string `gorm:"column:column_name" json:"columnName"`
	ColumnComment string `gorm:"column:column_comment" json:"columnComment"`
	ColumnType    string `gorm:"column:column_type" json:"columnType"`
	JavaType      string `gorm:"column:java_type" json:"javaType"`
	JavaField     string `gorm:"column:java_field" json:"javaField"`
	IsPk          string `gorm:"column:is_pk" json:"isPk"`
	IsIncrement   string `gorm:"column:is_increment" json:"isIncrement"`
	IsRequired    string `gorm:"column:is_required" json:"isRequired"`
	IsInsert      string `gorm:"column:is_insert" json:"isInsert"`
	IsEdit        string `gorm:"column:is_edit" json:"isEdit"`
	IsList        string `gorm:"column:is_list" json:"isList"`
	IsQuery       string `gorm:"column:is_query" json:"isQuery"`
	QueryType     string `gorm:"column:query_type" json:"queryType"`
	HTMLType      string `gorm:"column:html_type" json:"htmlType"`
	DictType      string `gorm:"column:dict_type" json:"dictType"`
	Sort          *int   `gorm:"column:sort" json:"sort"`
}

// TableName 表名。
func (GenTableColumn) TableName() string { return "gen_table_column" }
