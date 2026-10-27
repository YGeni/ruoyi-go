package repository

import (
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"gorm.io/gorm"
)

// deptListQuery 构造 selectDeptList 的基础查询（SysDeptMapper.xml:35-49）。
//
// Java：
//
//	select d.* from sys_dept d
//	where d.del_flag = '0' [and dept_id=?] [and parent_id=?] [and dept_name like ?] [and status=?]
//	${params.dataScope}
//	order by d.parent_id, d.order_num
//
// 别名 d 是硬约束：SysDeptServiceImpl.java:45 的 @DataScope(deptAlias = "d")
// 生成的条件引用 d.dept_id。userAlias 取注解默认值 ""（DataScope.java:36），
// 故数据范围 5（仅本人）在这里退化为 d.dept_id = 0。
func deptListQuery(q *query.DeptList, scope func(*gorm.DB) *gorm.DB) *gorm.DB {
	tx := db.DB.Table("sys_dept d").
		Select("d.*").
		Where("d.del_flag = ?", "0")
	if q.DeptID != 0 {
		tx = tx.Where("d.dept_id = ?", q.DeptID)
	}
	if q.ParentID != 0 {
		tx = tx.Where("d.parent_id = ?", q.ParentID)
	}
	if q.DeptName != "" {
		tx = tx.Where("d.dept_name LIKE ?", "%"+q.DeptName+"%")
	}
	if q.Status != "" {
		tx = tx.Where("d.status = ?", q.Status)
	}
	if scope != nil {
		tx = tx.Scopes(scope)
	}
	return tx
}

// SelectDeptList 分页查询部门，等价 SysDeptServiceImpl.java:45 的
// @DataScope(deptAlias = "d") + SysDeptMapper.xml:35-49。
//
// 排序固定 d.parent_id, d.order_num（Java 写死在 SQL 末尾）。
func SelectDeptList(q *query.DeptList, p *page.PageDomain, scope func(*gorm.DB) *gorm.DB) ([]entity.SysDept, int64, error) {
	var depts []entity.SysDept
	tx := deptListQuery(q, scope).Order("d.parent_id, d.order_num")
	total, err := page.Paginate(tx, p, "d", "", &depts)
	if err != nil {
		return nil, 0, err
	}
	return depts, total, nil
}

// SelectDeptByID 按主键查部门，等价 SysDeptMapper.xml selectDeptById。
//
// DIFF(review): Java 的 selectDeptById 不带 del_flag 过滤，此处保持一致。
func SelectDeptByID(deptID int64) (*entity.SysDept, error) {
	var d entity.SysDept
	err := db.DB.Where("dept_id = ?", deptID).First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// CheckDeptNameUnique 同父部门下部门名是否可用，
// 等价 SysDeptMapper.xml checkDeptNameUnique(deptName, parentId)。
func CheckDeptNameUnique(deptName string, parentID, excludeDeptID int64) (bool, error) {
	tx := db.DB.Table("sys_dept").
		Where("dept_name = ?", deptName).
		Where("parent_id = ?", parentID).
		Where("del_flag = ?", "0")
	if excludeDeptID != 0 {
		tx = tx.Where("dept_id != ?", excludeDeptID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n == 0, nil
}

// HasChildByDeptID 是否存在子部门，等价 SysDeptMapper.xml hasChildByDeptId。
func HasChildByDeptID(deptID int64) (bool, error) {
	var n int64
	err := db.DB.Table("sys_dept").Where("parent_id = ?", deptID).Where("del_flag = ?", "0").Count(&n).Error
	return n > 0, err
}

// CountNormalChildrenDept 统计正常状态的子部门数，
// 等价 SysDeptMapper.xml selectNormalChildrenDeptById。
func CountNormalChildrenDept(deptID int64) (int64, error) {
	var n int64
	err := db.DB.Table("sys_dept").
		Where("parent_id = ?", deptID).
		Where("status = ?", "0").
		Where("del_flag = ?", "0").
		Count(&n).Error
	return n, err
}

// CheckDeptExistUser 部门下是否存在用户，等价 SysDeptMapper.xml checkDeptExistUser。
func CheckDeptExistUser(deptID int64) (bool, error) {
	var n int64
	err := db.DB.Table("sys_user").
		Where("dept_id = ?", deptID).
		Where("del_flag = ?", "0").
		Count(&n).Error
	return n > 0, err
}

// InsertDept 新增部门，等价 SysDeptMapper.xml insertDept（动态列）。
//
// DIFF(review): 本库 sys_dept **没有 remark 列**（DESC sys_dept 无该字段），
// 而 BaseModel.Remark 是所有实体共用的，直接 Create 会报
// "Unknown column 'remark' in 'field list'"，故显式 Omit 掉。
// 前端提交到部门的 remark 因此被丢弃（Java 的 insertDept 也不含 remark）。
func InsertDept(d *entity.SysDept) error {
	// 同 InsertRole：Java 靠 DDL 默认值，Go 必须显式补，否则 del_flag/status 为空串
	if d.DelFlag == "" {
		d.DelFlag = model.DelFlagExists
	}
	if d.Status == "" {
		d.Status = config.Normal
	}
	return db.DB.Omit("remark").Create(d).Error
}

// UpdateDept 修改部门，等价 SysDeptMapper.xml updateDept 的 <set> 动态列。
//
// 显式列与 Java 的 if 分支一致：parent_id / dept_name / ancestors / order_num /
// leader / phone / email / status / update_by；update_time 恒更新。
func UpdateDept(d *entity.SysDept) error {
	sets := map[string]any{"update_time": time.Now()}
	if d.ParentID != 0 {
		sets["parent_id"] = d.ParentID
	}
	if d.DeptName != "" {
		sets["dept_name"] = d.DeptName
	}
	if d.Ancestors != "" {
		sets["ancestors"] = d.Ancestors
	}
	if d.OrderNum != 0 {
		sets["order_num"] = d.OrderNum
	}
	if d.Leader != "" {
		sets["leader"] = d.Leader
	}
	if d.Phone != "" {
		sets["phone"] = d.Phone
	}
	if d.Email != "" {
		sets["email"] = d.Email
	}
	if d.Status != "" {
		sets["status"] = d.Status
	}
	if d.UpdateBy != "" {
		sets["update_by"] = d.UpdateBy
	}
	return db.DB.Table("sys_dept").Where("dept_id = ?", d.DeptID).Updates(sets).Error
}

// UpdateDeptChildren 批量修改子部门的 ancestors，
// 等价 SysDeptMapper.xml updateDeptChildren 的 case-when。
//
// DIFF(review): Java 用一条 case-when 批量更新；Go 侧按每个部门一条 UPDATE，
// 语义等价（部门树的子节点数量有限），可读性更好。
func UpdateDeptChildren(depts []entity.SysDept) error {
	if len(depts) == 0 {
		return nil
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		for i := range depts {
			if err := tx.Table("sys_dept").Where("dept_id = ?", depts[i].DeptID).
				Updates(map[string]any{"ancestors": depts[i].Ancestors, "update_time": time.Now()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateDeptSort 只改排序号，等价 SysDeptMapper.xml updateDeptSort
// （SysDeptController.updateSort 专用，Java 侧**不更新 update_time**）。
func UpdateDeptSort(deptID int64, orderNum int) error {
	return db.DB.Table("sys_dept").Where("dept_id = ?", deptID).
		Updates(map[string]any{"order_num": orderNum}).Error
}

// UpdateDeptStatusNormal 把一批部门置为正常状态，
// 等价 SysDeptMapper.xml updateDeptStatusNormal（删除父部门后启用子部门）。
func UpdateDeptStatusNormal(deptIDs []int64) error {
	if len(deptIDs) == 0 {
		return nil
	}
	return db.DB.Table("sys_dept").Where("dept_id IN ?", deptIDs).
		Updates(map[string]any{"status": "0", "update_time": time.Now()}).Error
}

// DeleteDeptByID 逻辑删除部门，等价 SysDeptMapper.xml deleteDeptById：del_flag = '2'。
func DeleteDeptByID(deptID int64) error {
	return db.DB.Table("sys_dept").Where("dept_id = ?", deptID).
		Updates(map[string]any{"del_flag": "2", "update_time": time.Now()}).Error
}
