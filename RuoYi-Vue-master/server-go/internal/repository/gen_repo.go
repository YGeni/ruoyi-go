package repository

import (
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"gorm.io/gorm"
)

// ---------- gen_table ----------

// SelectGenTableList 分页查询（业务表），等价 GenTableMapper.xml 的 selectGenTableList。
func SelectGenTableList(q *query.GenTableList, p *page.PageDomain) ([]entity.GenTable, int64, error) {
	qb := db.DB.Model(&entity.GenTable{})
	if q != nil {
		if q.TableName != "" {
			qb = qb.Where("TABLE_NAME like ?", "%"+q.TableName+"%")
		}
		if q.TableComment != "" {
			qb = qb.Where("TABLE_COMMENT like ?", "%"+q.TableComment+"%")
		}
		if q.BeginTime != "" {
			qb = qb.Where("create_time >= ?", q.BeginTime)
		}
		if q.EndTime != "" {
			qb = qb.Where("create_time <= ?", q.EndTime)
		}
	}
	var rows []entity.GenTable
	total, err := page.Paginate(qb, p, "", "*", &rows)
	return rows, total, err
}

// SelectGenTableByID 查业务表 + 其列。
//
// Java 侧是 GenTableServiceImpl.selectGenTableById（L 附近）：先查 gen_table，
// 再 setTableFromOptions 把 options 反序列化，最后查 gen_table_column 挂到 columns。
func SelectGenTableByID(tableID int64) (*entity.GenTable, error) {
	var t entity.GenTable
	if err := db.DB.Where("table_id = ?", tableID).First(&t).Error; err != nil {
		return nil, err
	}
	cols, err := SelectGenTableColumns(tableID)
	if err != nil {
		return nil, err
	}
	t.Columns = cols
	for i := range cols {
		if cols[i].IsPk == "1" {
			c := cols[i]
			t.PkColumn = &c
			break
		}
	}
	return &t, nil
}

// SelectGenTableByName 按表名查，给 genCode 用（Java 的 selectGenTableByName）。
func SelectGenTableByName(tableName string) (*entity.GenTable, error) {
	var t entity.GenTable
	if err := db.DB.Where("table_name = ?", tableName).First(&t).Error; err != nil {
		return nil, err
	}
	cols, err := SelectGenTableColumns(t.TableID)
	if err != nil {
		return nil, err
	}
	t.Columns = cols
	for i := range cols {
		if cols[i].IsPk == "1" {
			c := cols[i]
			t.PkColumn = &c
			break
		}
	}
	return &t, nil
}

// SelectGenTableColumns 查某张业务表的全部列（按 sort 升序）。
func SelectGenTableColumns(tableID int64) ([]entity.GenTableColumn, error) {
	var rows []entity.GenTableColumn
	err := db.DB.Where("table_id = ?", tableID).Order("sort asc").Find(&rows).Error
	return rows, err
}

// InsertGenTable 新增业务表，返回自增主键（后续要拿它插列）。
func InsertGenTable(t *entity.GenTable) error {
	now := time.Now()
	t.CreateTime = &now
	return db.DB.Create(t).Error
}

// UpdateGenTable 修改业务表。
func UpdateGenTable(t *entity.GenTable) error {
	now := time.Now()
	t.UpdateTime = &now
	return db.DB.Where("table_id = ?", t.TableID).
		Select("table_name", "table_comment", "sub_table_name", "sub_table_fk_name",
			"class_name", "tpl_category", "tpl_web_type", "package_name", "module_name",
			"business_name", "function_name", "function_author", "form_col_num", "gen_type",
			"gen_path", "options", "remark", "update_by", "update_time").
		Updates(t).Error
}

// DeleteGenTableByIDs 批量删除业务表及其列。
//
// 等价 GenTableServiceImpl.deleteGenTableByIds：先删列再删表（同一个事务）。
func DeleteGenTableByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("table_id IN ?", ids).Delete(&entity.GenTableColumn{}).Error; err != nil {
			return err
		}
		return tx.Where("table_id IN ?", ids).Delete(&entity.GenTable{}).Error
	})
}

// UpdateGenTableColumns 批量更新列（前端编辑后提交整列集合）。
//
// 等价 GenTableMapper.xml 的 updateGenTableColumn：按 column_id 更新。
func UpdateGenTableColumns(cols []entity.GenTableColumn) error {
	if len(cols) == 0 {
		return nil
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		for i := range cols {
			c := cols[i]
			if c.ColumnID == 0 {
				continue
			}
			if err := tx.Where("column_id = ?", c.ColumnID).
				Select("column_comment", "java_type", "java_field", "is_pk", "is_increment",
					"is_required", "is_insert", "is_edit", "is_list", "is_query", "query_type",
					"html_type", "dict_type", "sort").
				Updates(&c).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SelectGenTableColumnsByNames 按列名集合查（synchDb 时用于保留旧配置）。
func SelectGenTableColumnsByNames(tableID int64, names []string) ([]entity.GenTableColumn, error) {
	var rows []entity.GenTableColumn
	err := db.DB.Where("table_id = ? AND column_name IN ?", tableID, names).Find(&rows).Error
	return rows, err
}

// ReplaceGenTableColumns 用新列集合替换旧列（synchDb / importTable 后写回）。
//
// Java 的 GenTableMapper.batchInsertGenTableColumn 是"插入"，
// 同步时靠的是先 deleteGenTableColumns 再插入，这里合成一步。
func ReplaceGenTableColumns(tableID int64, cols []entity.GenTableColumn) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("table_id = ?", tableID).Delete(&entity.GenTableColumn{}).Error; err != nil {
			return err
		}
		if len(cols) == 0 {
			return nil
		}
		for i := range cols {
			cols[i].ColumnID = 0
			cols[i].TableID = tableID
		}
		return tx.CreateInBatches(cols, 100).Error
	})
}

// ---------- information_schema：对应 GenTableMapper.xml 的 db 相关查询 ----------

// DbTable 数据库里的一张物理表（来自 information_schema.tables）。
type DbTable struct {
	TableName    string `gorm:"column:table_name" json:"tableName"`
	TableComment string `gorm:"column:table_comment" json:"tableComment"`
	CreateTime   string `gorm:"column:create_time" json:"createTime"`
}

// DbColumn 数据库里的一列（来自 information_schema.columns）。
type DbColumn struct {
	ColumnName    string `gorm:"column:column_name" json:"columnName"`
	ColumnComment string `gorm:"column:column_comment" json:"columnComment"`
	ColumnType    string `gorm:"column:column_type" json:"columnType"`
	ColumnKey     string `gorm:"column:column_key" json:"columnKey"`
	Extra         string `gorm:"column:extra" json:"extra"`
	IsNullable    string `gorm:"column:is_nullable" json:"isNullable"`
}

// SelectDbTableList 查数据库里**尚未导入**的表，
// 等价 GenTableMapper.xml 的 selectDbTableList（排除已导入 + 排除定时任务表）。
func SelectDbTableList(q *query.GenTableList, p *page.PageDomain) ([]DbTable, int64, error) {
	// information_schema 的列名在 MySQL 里是**大写**的（TABLE_NAME），
	// GORM 按列名匹配 tag 时会匹配不上 -> 字段全空，所以必须起别名。
	qb := db.DB.Table("information_schema.tables").
		Select("TABLE_NAME AS table_name, TABLE_COMMENT AS table_comment, CREATE_TIME AS create_time").
		Where("table_schema = (SELECT DATABASE())").
		Where("TABLE_NAME NOT LIKE 'qr%'").
		Where("TABLE_NAME NOT LIKE 'qrtz%'").
		Where("TABLE_NAME NOT IN (SELECT table_name FROM gen_table)")
	if q != nil {
		if q.TableName != "" {
			qb = qb.Where("TABLE_NAME like ?", "%"+q.TableName+"%")
		}
		if q.TableComment != "" {
			qb = qb.Where("TABLE_COMMENT like ?", "%"+q.TableComment+"%")
		}
	}
	var rows []DbTable
	total, err := page.Paginate(qb, p, "", "*", &rows)
	return rows, total, err
}

// SelectDbTableByNames 按表名集合查物理表（importTable 用）。
func SelectDbTableByNames(names []string) ([]DbTable, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var rows []DbTable
	err := db.DB.Table("information_schema.tables").
		Select("TABLE_NAME AS table_name, TABLE_COMMENT AS table_comment, CREATE_TIME AS create_time").
		Where("table_schema = (SELECT DATABASE())").
		Where("TABLE_NAME IN ?", names).
		Find(&rows).Error
	return rows, err
}

// SelectDbColumns 查某张物理表的列，按 ordinal_position 升序。
func SelectDbColumns(tableName string) ([]DbColumn, error) {
	var rows []DbColumn
	err := db.DB.Table("information_schema.columns").
		Select("COLUMN_NAME AS column_name, COLUMN_COMMENT AS column_comment, COLUMN_TYPE AS column_type, "+
			"COLUMN_KEY AS column_key, EXTRA AS extra, IS_NULLABLE AS is_nullable").
		Where("table_schema = (SELECT DATABASE())").
		Where("TABLE_NAME = ?", tableName).
		Order("ordinal_position asc").
		Find(&rows).Error
	return rows, err
}

// ExecSQL 执行建表语句，等价 GenController.createTableSave 里的
// genTableService.createTable(sql)（直接 Execute 原始 SQL）。
func ExecSQL(sql string) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return nil
	}
	return db.DB.Exec(sql).Error
}
