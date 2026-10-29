package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"github.com/gin-gonic/gin"
)

// SelectDeptList 部门列表，等价
//
//	@DataScope(deptAlias = "d")
//	public List<SysDept> selectDeptList(SysDept dept)
//
// **DataScope 五个落点之一**（deptAlias="d"、userAlias 取注解默认空串）。
// 部门表本身就是数据范围的作用对象，因此本落点的语义是"能看到哪些部门"。
func SelectDeptList(c *gin.Context, lu *entity.LoginUser, q *query.DeptList, p *page.PageDomain) ([]entity.SysDept, int64, error) {
	return repository.SelectDeptList(q, p, DataScope(c, lu, "d", ""))
}

// SelectDeptByID 按主键查部门，等价 SysDeptServiceImpl.selectDeptById(L117-121)。
func SelectDeptByID(deptID int64) (*entity.SysDept, error) {
	return repository.SelectDeptByID(deptID)
}

// CheckDeptNameUnique 同父部门下部门名是否唯一，
// 等价 SysDeptServiceImpl.checkDeptNameUnique(L165-178)。
func CheckDeptNameUnique(d *entity.SysDept) (bool, error) {
	return repository.CheckDeptNameUnique(d.DeptName, d.ParentID, d.DeptID)
}

// InsertDept 新增部门，等价 SysDeptServiceImpl.insertDept(L186-199)。
//
// 校验顺序与 Java 一致：
//  1. 父部门必须存在且为正常状态（UserConstants.DEPT_NORMAL）
//  2. ancestors = 父部门 ancestors + "," + 父部门 deptId
func InsertDept(d *entity.SysDept) error {
	parent, err := repository.SelectDeptByID(d.ParentID)
	if err != nil {
		return err
	}
	if parent == nil {
		return errors.New("上级部门不存在")
	}
	if parent.Status != config.Normal {
		return errors.New("部门停用，不允许新增")
	}
	if parent.Ancestors == "" {
		d.Ancestors = fmt.Sprintf("0,%d", parent.DeptID)
	} else {
		d.Ancestors = parent.Ancestors + "," + fmt.Sprint(parent.DeptID)
	}
	return repository.InsertDept(d)
}

// UpdateDept 修改部门，等价 SysDeptServiceImpl.updateDept(L206-241)。
//
// Java 的关键逻辑：
//   - 新父部门存在且状态正常，否则"部门停用，不允许启用"
//   - ancestors = 新父部门 ancestors + "," + 新父部门 deptId
//   - 若父部门变更，同步更新**全部子节点**的 ancestors
//     （子节点 ancestors 前缀替换：oldAncestors + "," + deptId -> newAncestors）
//   - 若新状态为正常而旧状态为停用，则把所有子节点一并置为正常
//
// DIFF(review): Java 的 updateDeptChildren 用一条 case-when 批量更新；
// Go 侧改为事务内逐条更新（子节点数量有限），语义等价。
func UpdateDept(d *entity.SysDept) error {
	oldDept, err := repository.SelectDeptByID(d.DeptID)
	if err != nil {
		return err
	}
	if oldDept == nil {
		return errors.New("部门不存在")
	}
	newParent, err := repository.SelectDeptByID(d.ParentID)
	if err != nil {
		return err
	}
	if newParent == nil {
		return errors.New("上级部门不存在")
	}
	if newParent.Status != config.Normal {
		return errors.New("部门停用，不允许启用")
	}

	// 计算新的 ancestors
	newAncestors := d.Ancestors
	if newAncestors == "" {
		if newParent.Ancestors == "" {
			newAncestors = fmt.Sprintf("0,%d", newParent.DeptID)
		} else {
			newAncestors = newParent.Ancestors + "," + fmt.Sprint(newParent.DeptID)
		}
	}
	oldAncestors := oldDept.Ancestors
	if oldAncestors == "" {
		oldAncestors = fmt.Sprintf("0,%d", oldDept.ParentID)
	}
	d.Ancestors = newAncestors

	if err := repository.UpdateDept(d); err != nil {
		return err
	}

	// 父部门变更 -> 同步子树 ancestors
	if newAncestors != oldAncestors {
		children, err := childrenOfDept(d.DeptID)
		if err != nil {
			return err
		}
		updates := make([]entity.SysDept, 0, len(children))
		for i := range children {
			updates = append(updates, entity.SysDept{
				DeptID:    children[i].DeptID,
				Ancestors: strings.Replace(children[i].Ancestors, oldAncestors, newAncestors, 1),
			})
		}
		if err := repository.UpdateDeptChildren(updates); err != nil {
			return err
		}
	}

	// 状态由停用改为正常 -> 子节点一并正常
	if d.Status == config.Normal && oldDept.Status == config.Disable {
		children, err := childrenOfDept(d.DeptID)
		if err != nil {
			return err
		}
		ids := make([]int64, 0, len(children))
		for i := range children {
			ids = append(ids, children[i].DeptID)
		}
		return repository.UpdateDeptStatusNormal(ids)
	}
	return nil
}

// childrenOfDept 取某部门的直接子部门（含停用），等价 SysDeptMapper.xml 的
// selectChildrenDeptById（updateDept 的子树 ancestors 重算需要）。
func childrenOfDept(deptID int64) ([]entity.SysDept, error) {
	var rows []entity.SysDept
	err := db.DB.Table("sys_dept").Where("parent_id = ?", deptID).Find(&rows).Error
	return rows, err
}

// UpdateDeptSort 修改部门排序号（/system/dept/updateSort），
// 等价 SysDeptController.updateSort + SysDeptMapper.xml updateDeptSort。
//
// DIFF(review): Java 侧 updateSort 不更新 update_time，Go 侧保持一致。
func UpdateDeptSort(deptID int64, orderNum int) error {
	return repository.UpdateDeptSort(deptID, orderNum)
}

// DeleteDeptByID 删除部门，等价 SysDeptController.remove(L90-104) 的前置校验
// + SysDeptServiceImpl.deleteDeptById(L250-255)。
//
// 前置校验（顺序与 Java 一致）：存在下级部门 -> 部门存在用户 -> 删除。
func DeleteDeptByID(deptID int64) error {
	hasChild, err := repository.HasChildByDeptID(deptID)
	if err != nil {
		return err
	}
	if hasChild {
		return errors.New("存在下级部门,不允许删除")
	}
	existUser, err := repository.CheckDeptExistUser(deptID)
	if err != nil {
		return err
	}
	if existUser {
		return errors.New("部门存在用户,不允许删除")
	}
	return repository.DeleteDeptByID(deptID)
}

// CheckDeptDataScope 校验部门数据权限，等价 SysDeptServiceImpl.checkDeptDataScope(L179-190)。
//
// Java：非超管用户如果对目标部门 selectDeptList 查不到结果，则抛"没有权限访问部门数据！"。
func CheckDeptDataScope(c *gin.Context, lu *entity.LoginUser, deptID int64) error {
	if lu == nil || lu.IsAdmin() || deptID == 0 {
		return nil
	}
	rows, _, err := SelectDeptList(c, lu, &query.DeptList{DeptID: deptID},
		&page.PageDomain{PageNum: 1, PageSize: 1})
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return errors.New("没有权限访问部门数据！")
	}
	return nil
}
