// Package repository 对应 Java 侧的 Mapper 层（GORM 实现，全部参数化，禁止拼接 SQL）。
package repository

import (
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"

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
