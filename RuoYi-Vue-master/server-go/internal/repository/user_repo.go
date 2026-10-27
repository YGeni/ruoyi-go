// Package repository 对应 Java 侧的 Mapper 层（GORM 实现，全部参数化，禁止拼接 SQL）。
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

// SelectUserByUserName 按登录名查询用户，等价于 SysUserMapper.xml:124-127 的
// selectUserByUserName：where u.user_name = #{userName} and u.del_flag = '0'
//
// association(dept) 与 collection(roles)（SysUserMapper.xml:27-28）
// 用 Preload 实现，等价于 resultMap 的嵌套映射。
func SelectUserByUserName(userName string) (*entity.SysUser, error) {
	var u entity.SysUser
	err := db.DB.Preload("Dept").Preload("Roles").
		Where("user_name = ?", userName).
		Where("del_flag = ?", model.DelFlagExists).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// userListQuery 构造 selectUserList 的基础查询（SysUserMapper.xml:59-83）。
//
// SQL 形状与 Java 完全一致：
//
//	select ... from sys_user u left join sys_dept d on u.dept_id = d.dept_id
//	where u.del_flag = '0' [and ...] ${params.dataScope}
//
// 别名 u / d 是硬约束：@DataScope(deptAlias="d", userAlias="u") 生成的条件
// 直接引用 d.dept_id / u.user_id（SysUserServiceImpl.java:76/89/102）。
//
// scope 为数据权限 Scope（admin 直通时不施加任何条件）；nil 表示不过滤。
func userListQuery(q *query.UserList, scope func(*gorm.DB) *gorm.DB) *gorm.DB {
	tx := db.DB.Table("sys_user u").
		Select("u.*").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		Where("u.del_flag = ?", model.DelFlagExists)
	if q.UserID != 0 {
		tx = tx.Where("u.user_id = ?", q.UserID)
	}
	if q.UserName != "" {
		tx = tx.Where("u.user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.Status != "" {
		tx = tx.Where("u.status = ?", q.Status)
	}
	if q.Phonenumber != "" {
		tx = tx.Where("u.phonenumber LIKE ?", "%"+q.Phonenumber+"%")
	}
	if q.BeginTime != "" {
		tx = tx.Where("date_format(u.create_time,'%Y%m%d') >= date_format(?,'%Y%m%d')", q.BeginTime)
	}
	if q.EndTime != "" {
		tx = tx.Where("date_format(u.create_time,'%Y%m%d') <= date_format(?,'%Y%m%d')", q.EndTime)
	}
	if q.DeptID != 0 {
		// find_in_set 参数化，禁止字符串拼接（AGENTS.md 全局规则 4）
		tx = tx.Where("(u.dept_id = ? OR u.dept_id IN (SELECT t.dept_id FROM sys_dept t WHERE find_in_set(?, t.ancestors)))",
			q.DeptID, q.DeptID)
	}
	if q.SearchValue != "" {
		v := "%" + q.SearchValue + "%"
		tx = tx.Where("(u.user_name LIKE ? OR u.nick_name LIKE ?)", v, v)
	}
	if scope != nil {
		tx = tx.Scopes(scope)
	}
	return tx
}

// attachDepts 回填每行的 dept（等价 SysUserResult 的 association dept）。
//
// DIFF(review): Java 用一条 join SQL 同时取 d.dept_name/d.leader 并由 resultMap 组装；
// Go 侧拆成主查询 + 一次按 dept_id IN 的批量回填，避免 u.* 与 d.* 同名列（dept_id /
// create_time 等）在 Scan 时互相覆盖。返回给前端的字段与 Java 一致。
func attachDepts(users []entity.SysUser) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(users))
	seen := make(map[int64]struct{}, len(users))
	for i := range users {
		if users[i].DeptID == nil {
			continue
		}
		if _, ok := seen[*users[i].DeptID]; ok {
			continue
		}
		seen[*users[i].DeptID] = struct{}{}
		ids = append(ids, *users[i].DeptID)
	}
	if len(ids) == 0 {
		return nil
	}
	var depts []entity.SysDept
	if err := db.DB.Where("dept_id IN ?", ids).Find(&depts).Error; err != nil {
		return err
	}
	m := make(map[int64]*entity.SysDept, len(depts))
	for i := range depts {
		m[depts[i].DeptID] = &depts[i]
	}
	for i := range users {
		if users[i].DeptID == nil {
			continue
		}
		users[i].Dept = m[*users[i].DeptID]
	}
	return nil
}

// SelectUserList 分页查询用户列表，等价于 SysUserServiceImpl.java:76 的
// @DataScope(deptAlias="d", userAlias="u") + SysUserMapper.xml:59-83。
func SelectUserList(q *query.UserList, p *page.PageDomain, scope func(*gorm.DB) *gorm.DB) ([]entity.SysUser, int64, error) {
	var users []entity.SysUser
	total, err := page.Paginate(userListQuery(q, scope), p, "u", "", &users)
	if err != nil {
		return nil, 0, err
	}
	if err := attachDepts(users); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// AllocatedListQuery 暴露 selectAllocatedList 的查询构造器。
//
// 存在的意义：单测要用 DryRun 断言该查询的 SQL 形状（它有两个额外 join，
// 且别名组合与 selectUserList 不同），而 SelectAllocatedList 内部用的是
// 包级 db.DB，测试拿不到最终的 Statement。
func AllocatedListQuery(roleID int64, q *query.AuthUserList, scope func(*gorm.DB) *gorm.DB) *gorm.DB {
	tx := db.DB.Table("sys_user u").
		Select("DISTINCT u.*").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		Joins("LEFT JOIN sys_user_role ur ON u.user_id = ur.user_id").
		Joins("LEFT JOIN sys_role r ON r.role_id = ur.role_id").
		Where("u.del_flag = ?", model.DelFlagExists).
		Where("r.role_id = ?", roleID)
	tx = applyAuthUserFilters(tx, q)
	if scope != nil {
		tx = tx.Scopes(scope)
	}
	return tx
}

// SelectAllocatedList 查询已分配该角色的用户，等价于 SysUserMapper.xml:85-102。
//
// 原 SQL 为 select distinct + 四表 left join，@DataScope(deptAlias="d", userAlias="u")。
func SelectAllocatedList(roleID int64, q *query.AuthUserList, p *page.PageDomain, scope func(*gorm.DB) *gorm.DB) ([]entity.SysUser, int64, error) {
	tx := AllocatedListQuery(roleID, q, scope)
	var users []entity.SysUser
	total, err := page.Paginate(tx, p, "u", "DISTINCT u.user_id", &users)
	if err != nil {
		return nil, 0, err
	}
	if err := attachDepts(users); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UnallocatedListQuery 暴露 selectUnallocatedList 的查询构造器，用途同 AllocatedListQuery。
func UnallocatedListQuery(roleID int64, q *query.AuthUserList, scope func(*gorm.DB) *gorm.DB) *gorm.DB {
	tx := db.DB.Table("sys_user u").
		Select("DISTINCT u.*").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		Joins("LEFT JOIN sys_user_role ur ON u.user_id = ur.user_id").
		Joins("LEFT JOIN sys_role r ON r.role_id = ur.role_id").
		Where("u.del_flag = ?", model.DelFlagExists).
		Where("(r.role_id != ? OR r.role_id IS NULL)", roleID).
		Where("u.user_id NOT IN (SELECT su.user_id FROM sys_user su INNER JOIN sys_user_role sur ON su.user_id = sur.user_id AND sur.role_id = ?)", roleID)
	tx = applyAuthUserFilters(tx, q)
	if scope != nil {
		tx = tx.Scopes(scope)
	}
	return tx
}

// SelectUnallocatedList 查询未分配该角色的用户，等价于 SysUserMapper.xml:103-121。
func SelectUnallocatedList(roleID int64, q *query.AuthUserList, p *page.PageDomain, scope func(*gorm.DB) *gorm.DB) ([]entity.SysUser, int64, error) {
	tx := UnallocatedListQuery(roleID, q, scope)
	var users []entity.SysUser
	total, err := page.Paginate(tx, p, "u", "DISTINCT u.user_id", &users)
	if err != nil {
		return nil, 0, err
	}
	if err := attachDepts(users); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func applyAuthUserFilters(tx *gorm.DB, q *query.AuthUserList) *gorm.DB {
	if q.UserName != "" {
		tx = tx.Where("u.user_name LIKE ?", "%"+q.UserName+"%")
	}
	if q.Phonenumber != "" {
		tx = tx.Where("u.phonenumber LIKE ?", "%"+q.Phonenumber+"%")
	}
	return tx
}

// SelectUserByID 按主键查用户（含 dept / roles），等价于 SysUserMapper.xml:129-131。
//
// DIFF(review): Java 的 selectUserById 不带 del_flag 过滤，此处保持同款不过滤；
// 需要过滤的地方由调用方（如登录）自行处理。
func SelectUserByID(userID int64) (*entity.SysUser, error) {
	var u entity.SysUser
	err := db.DB.Preload("Dept").Preload("Roles").
		Where("user_id = ?", userID).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SelectRoleIDsByUserID 查用户已分配的角色 ID，等价 SysRoleMapper.xml 的 selectRolesByUserId 取 id。
func SelectRoleIDsByUserID(userID int64) ([]int64, error) {
	var ids []int64
	err := db.DB.Table("sys_user_role").Where("user_id = ?", userID).Pluck("role_id", &ids).Error
	return ids, err
}

// CheckUserNameUnique 登录名是否已存在，等价 SysUserMapper.xml:133-135。
// 返回 true 表示可用（不重复）。
func CheckUserNameUnique(userName string, excludeUserID int64) (bool, error) {
	return notExists(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("user_name = ?", userName)
	}, excludeUserID)
}

// CheckPhoneUnique 手机号是否已存在，等价 SysUserMapper.xml:137-139。
func CheckPhoneUnique(phone string, excludeUserID int64) (bool, error) {
	if phone == "" {
		return true, nil
	}
	return notExists(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("phonenumber = ?", phone)
	}, excludeUserID)
}

// CheckEmailUnique 邮箱是否已存在，等价 SysUserMapper.xml:141-143。
func CheckEmailUnique(email string, excludeUserID int64) (bool, error) {
	if email == "" {
		return true, nil
	}
	return notExists(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("email = ?", email)
	}, excludeUserID)
}

func notExists(cond func(*gorm.DB) *gorm.DB, excludeUserID int64) (bool, error) {
	tx := cond(db.DB.Table("sys_user")).Where("del_flag = ?", model.DelFlagExists)
	if excludeUserID != 0 {
		// Java 的 checkXxxUnique 语义：命中自己时不算重复
		tx = tx.Where("user_id != ?", excludeUserID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n == 0, nil
}

// InsertUser 新增用户，等价 SysUserMapper.xml:145-178。
//
// 同 InsertRole：显式补 del_flag/status 默认值，避免 Create 写入空串零值。
func InsertUser(u *entity.SysUser) error {
	if u.DelFlag == "" {
		u.DelFlag = model.DelFlagExists
	}
	if u.Status == "" {
		u.Status = config.Normal
	}
	return db.DB.Create(u).Error
}

// UpdateUser 修改用户，等价 SysUserMapper.xml:180-201 的 <set> 动态列
// （只更新非空字段，update_time 恒更新）。
func UpdateUser(u *entity.SysUser) error {
	sets := map[string]any{}
	if u.DeptID != nil && *u.DeptID != 0 {
		sets["dept_id"] = *u.DeptID
	}
	if u.NickName != "" {
		sets["nick_name"] = u.NickName
	}
	if u.Email != "" {
		sets["email"] = u.Email
	}
	if u.Phonenumber != "" {
		sets["phonenumber"] = u.Phonenumber
	}
	if u.Sex != "" {
		sets["sex"] = u.Sex
	}
	if u.Avatar != "" {
		sets["avatar"] = u.Avatar
	}
	if u.Password != "" {
		sets["password"] = u.Password
	}
	if u.Status != "" {
		sets["status"] = u.Status
	}
	if u.LoginIP != "" {
		sets["login_ip"] = u.LoginIP
	}
	if u.LoginDate != nil {
		sets["login_date"] = *u.LoginDate
	}
	if u.UpdateBy != "" {
		sets["update_by"] = u.UpdateBy
	}
	if u.Remark != "" {
		sets["remark"] = u.Remark
	}
	if len(sets) == 0 {
		return nil
	}
	// DIFF(review): 用 map 更新时 GORM 不触发 BeforeUpdate hook，故显式写 update_time
	// （Java 侧由 update_time = sysdate() 负责）。
	sets["update_time"] = time.Now()
	return db.DB.Table("sys_user").Where("user_id = ?", u.UserID).Updates(sets).Error
}

// UpdateUserStatus 只改状态，等价 SysUserMapper.xml:203-205。
func UpdateUserStatus(userID int64, status string) error {
	return db.DB.Table("sys_user").Where("user_id = ?", userID).
		Updates(map[string]any{"status": status, "update_time": time.Now()}).Error
}

// UpdateUserAvatar 只改头像，等价 SysUserMapper.xml:207-209（profile 头像用）。
func UpdateUserAvatar(userName, avatar string) error {
	return db.DB.Table("sys_user").Where("user_name = ?", userName).
		Updates(map[string]any{"avatar": avatar, "update_time": time.Now()}).Error
}

// ResetUserPwd 重置密码，等价 SysUserMapper.xml:216-218：
// 同时把 pwd_update_date 置为当前时间（影响 isDefaultModifyPwd / isPasswordExpired）。
func ResetUserPwd(userID int64, encrypted string) error {
	now := time.Now()
	return db.DB.Table("sys_user").Where("user_id = ?", userID).
		Updates(map[string]any{
			"password":        encrypted,
			"pwd_update_date": now,
			"update_time":     now,
		}).Error
}

// DeleteUserByIDs 逻辑删除，等价 SysUserMapper.xml:225-231：del_flag = '2'。
//
// 硬约束：不得使用 GORM 的 DeletedAt / 软删机制（del_flag 为 char(1) 三态，
// 见 model/base.go 的 DIFF 说明）。
func DeleteUserByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.DB.Table("sys_user").Where("user_id IN ?", ids).
		Updates(map[string]any{"del_flag": model.DelFlagDeleted, "update_time": time.Now()}).Error
}

// DeleteUserRole 删除用户的全部角色关联，等价 SysUserMapper.xml deleteUserRole。
func DeleteUserRole(userID int64) error {
	return db.DB.Where("user_id = ?", userID).Delete(&entity.SysUserRole{}).Error
}

// InsertUserRoles 批量写入用户角色关联，等价 SysUserMapper.xml batchUserRole。
func InsertUserRoles(userID int64, roleIDs []int64) error {
	if len(roleIDs) == 0 {
		return nil
	}
	rows := make([]entity.SysUserRole, 0, len(roleIDs))
	for _, rid := range roleIDs {
		rows = append(rows, entity.SysUserRole{UserID: userID, RoleID: rid})
	}
	return db.DB.Create(&rows).Error
}

// UpdateLoginInfo 更新最后登录 IP 与时间，等价于 SysLoginService.recordLoginInfo(L172-175)。
//
// DIFF(review): Java 侧 UPDATE 语句里 update_time 由 sysdate() 写入；
// 这里用 map 更新时 GORM 不会触发 BeforeUpdate hook，故显式带上 update_time。
func UpdateLoginInfo(userID int64, ip string, loginTime time.Time) error {
	return db.DB.Model(&entity.SysUser{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"login_ip":    ip,
			"login_date":  loginTime,
			"update_time": time.Now(),
		}).Error
}

// InsertLogininfor 写入登录日志 sys_logininfor。
//
// 本期为**同步最小实现**（已与需求方确认）：仅写
// user_name / ipaddr / status / msg / login_time 五个字段，
// browser / login_location / os 留空。
//
// DIFF(review): Java 侧由 AsyncManager 延迟 10ms 异步提交，并调用
// https://whois.pconline.com.cn/ipJson.jsp 解析归属地、用 UserAgentUtils 解析浏览器/系统
// （AsyncFactory.java:37-99、TokenService.setUserAgent L163-171）；
// 本期不引 UA 与归属地解析，相关字段留空，异步化与地址解析留到 1b。
func InsertLogininfor(rec *entity.SysLogininfor) error {
	return db.DB.Create(rec).Error
}
