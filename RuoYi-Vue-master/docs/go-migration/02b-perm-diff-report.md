# 1b 权限绑定双向 diff 报告

由 `internal/router/perm_diff_test.go` 自动生成于 `go test ./internal/router/`。
数据源：Java 控制器源码注解（运行时解析，非人工抄写）。

| 项 | 数量 |
|---|---|
| Java 端点总数（admin+generator+quartz） | 147 |
| 1b 范围内端点 | 104 |
| 二期延后端点 | 43 |
| 缺失（Java→Go） | 0 |
| 多余（Go→Java） | 0 |

## 缺失

（无）

## 多余

（无）

## 1b 范围内绑定清单

| Method | Path | 权限串 | Java 来源 |
|---|---|---|---|
| DELETE | `/monitor/cache/clearCacheAll` | monitor:cache:list | CacheController.java:114 |
| DELETE | `/monitor/cache/clearCacheKey/:cacheKey` | monitor:cache:list | CacheController.java:106 |
| DELETE | `/monitor/cache/clearCacheName/:cacheName` | monitor:cache:list | CacheController.java:97 |
| DELETE | `/monitor/logininfor/:infoIds` | monitor:logininfor:remove | SysLogininforController.java:57 |
| DELETE | `/monitor/logininfor/clean` | monitor:logininfor:remove | SysLogininforController.java:65 |
| DELETE | `/monitor/online/:tokenId` | monitor:online:forceLogout | SysUserOnlineController.java:75 |
| DELETE | `/monitor/operlog/:operIds` | monitor:operlog:remove | SysOperlogController.java:53 |
| DELETE | `/monitor/operlog/clean` | monitor:operlog:remove | SysOperlogController.java:61 |
| DELETE | `/system/config/:configIds` | system:config:remove | SysConfigController.java:113 |
| DELETE | `/system/config/refreshCache` | system:config:remove | SysConfigController.java:125 |
| DELETE | `/system/dept/:deptId` | system:dept:remove | SysDeptController.java:131 |
| DELETE | `/system/dict/data/:dictCodes` | system:dict:remove | SysDictDataController.java:113 |
| DELETE | `/system/dict/type/:dictIds` | system:dict:remove | SysDictTypeController.java:101 |
| DELETE | `/system/dict/type/refreshCache` | system:dict:remove | SysDictTypeController.java:113 |
| DELETE | `/system/menu/:menuId` | system:menu:remove | SysMenuController.java:150 |
| DELETE | `/system/notice/:noticeIds` | system:notice:remove | SysNoticeController.java:142 |
| DELETE | `/system/post/:postIds` | system:post:remove | SysPostController.java:112 |
| DELETE | `/system/role/:roleIds` | system:role:remove | SysRoleController.java:166 |
| DELETE | `/system/user/:userIds` | system:user:remove | SysUserController.java:177 |
| GET | `/common/download` | (仅需登录) | CommonController.java:34 |
| GET | `/common/download/resource` | (仅需登录) | CommonController.java:137 |
| GET | `/monitor/cache` | monitor:cache:list | CacheController.java:48 |
| GET | `/monitor/cache/getKeys/:cacheName` | monitor:cache:list | CacheController.java:80 |
| GET | `/monitor/cache/getNames` | monitor:cache:list | CacheController.java:73 |
| GET | `/monitor/cache/getValue/:cacheName/:cacheKey` | monitor:cache:list | CacheController.java:88 |
| GET | `/monitor/logininfor/list` | monitor:logininfor:list | SysLogininforController.java:32 |
| GET | `/monitor/logininfor/unlock/:userName` | monitor:logininfor:unlock | SysLogininforController.java:74 |
| GET | `/monitor/online/list` | monitor:online:list | SysUserOnlineController.java:35 |
| GET | `/monitor/operlog/list` | monitor:operlog:list | SysOperlogController.java:31 |
| GET | `/monitor/server` | monitor:server:list | ServerController.java:19 |
| GET | `/system/config/:configId` | system:config:query | SysConfigController.java:62 |
| GET | `/system/config/configKey/:configKey` | (仅需登录) | SysConfigController.java:72 |
| GET | `/system/config/list` | system:config:list | SysConfigController.java:34 |
| GET | `/system/dept/:deptId` | system:dept:query | SysDeptController.java:64 |
| GET | `/system/dept/list` | system:dept:list | SysDeptController.java:35 |
| GET | `/system/dept/list/exclude/:deptId` | system:dept:list | SysDeptController.java:52 |
| GET | `/system/dict/data/:dictCode` | system:dict:query | SysDictDataController.java:65 |
| GET | `/system/dict/data/list` | system:dict:list | SysDictDataController.java:37 |
| GET | `/system/dict/data/type/:dictType` | (仅需登录) | SysDictDataController.java:75 |
| GET | `/system/dict/type/:dictId` | system:dict:query | SysDictTypeController.java:59 |
| GET | `/system/dict/type/list` | system:dict:list | SysDictTypeController.java:34 |
| GET | `/system/dict/type/optionselect` | (仅需登录) | SysDictTypeController.java:125 |
| GET | `/system/menu/:menuId` | system:menu:query | SysMenuController.java:51 |
| GET | `/system/menu/list` | system:menu:list | SysMenuController.java:34 |
| GET | `/system/menu/roleMenuTreeselect/:roleId` | (仅需登录) | SysMenuController.java:71 |
| GET | `/system/menu/treeselect` | (仅需登录) | SysMenuController.java:61 |
| GET | `/system/notice/:noticeId` | (仅需登录) | SysNoticeController.java:56 |
| GET | `/system/notice/list` | system:notice:list | SysNoticeController.java:35 |
| GET | `/system/notice/listTop` | (仅需登录) | SysNoticeController.java:89 |
| GET | `/system/notice/readUsers/list` | system:notice:list | SysNoticeController.java:129 |
| GET | `/system/post/:postId` | system:post:query | SysPostController.java:62 |
| GET | `/system/post/list` | system:post:list | SysPostController.java:34 |
| GET | `/system/post/optionselect` | (仅需登录) | SysPostController.java:123 |
| GET | `/system/role/:roleId` | system:role:query | SysRoleController.java:78 |
| GET | `/system/role/authUser/allocatedList` | system:role:list | SysRoleController.java:187 |
| GET | `/system/role/authUser/unallocatedList` | system:role:list | SysRoleController.java:199 |
| GET | `/system/role/deptTree/:roleId` | system:role:query | SysRoleController.java:245 |
| GET | `/system/role/list` | system:role:list | SysRoleController.java:41 |
| GET | `/system/role/optionselect` | system:role:query | SysRoleController.java:177 |
| GET | `/system/user/` | system:user:query | SysUserController.java:100 |
| GET | `/system/user/:userId` | system:user:query | SysUserController.java:100 |
| GET | `/system/user/authRole/:userId` | system:user:query | SysUserController.java:221 |
| GET | `/system/user/deptTree` | system:user:list | SysUserController.java:250 |
| GET | `/system/user/list` | system:user:list | SysUserController.java:44 |
| GET | `/system/user/profile` | (仅需登录) | SysProfileController.java:38 |
| POST | `/common/upload` | (仅需登录) | CommonController.java:74 |
| POST | `/common/uploads` | (仅需登录) | CommonController.java:100 |
| POST | `/monitor/logininfor/export` | monitor:logininfor:export | SysLogininforController.java:47 |
| POST | `/monitor/operlog/export` | monitor:operlog:export | SysOperlogController.java:43 |
| POST | `/register` | (仅需登录) | SysRegisterController.java:22 |
| POST | `/system/config` | system:config:add | SysConfigController.java:81 |
| POST | `/system/dept` | system:dept:add | SysDeptController.java:75 |
| POST | `/system/dict/data` | system:dict:add | SysDictDataController.java:89 |
| POST | `/system/dict/type` | system:dict:add | SysDictTypeController.java:69 |
| POST | `/system/menu` | system:menu:add | SysMenuController.java:84 |
| POST | `/system/notice` | system:notice:add | SysNoticeController.java:65 |
| POST | `/system/notice/markRead` | (仅需登录) | SysNoticeController.java:104 |
| POST | `/system/notice/markReadAll` | (仅需登录) | SysNoticeController.java:116 |
| POST | `/system/post` | system:post:add | SysPostController.java:72 |
| POST | `/system/role` | system:role:add | SysRoleController.java:89 |
| POST | `/system/user` | system:user:add | SysUserController.java:122 |
| POST | `/system/user/profile/avatar` | (仅需登录) | SysProfileController.java:124 |
| POST | `/unlockscreen` | (仅需登录) | SysIndexController.java:43 |
| PUT | `/system/config` | system:config:edit | SysConfigController.java:97 |
| PUT | `/system/dept` | system:dept:edit | SysDeptController.java:91 |
| PUT | `/system/dept/updateSort` | system:dept:edit | SysDeptController.java:117 |
| PUT | `/system/dict/data` | system:dict:edit | SysDictDataController.java:101 |
| PUT | `/system/dict/type` | system:dict:edit | SysDictTypeController.java:85 |
| PUT | `/system/menu` | system:menu:edit | SysMenuController.java:108 |
| PUT | `/system/menu/updateSort` | system:menu:edit | SysMenuController.java:136 |
| PUT | `/system/notice` | system:notice:edit | SysNoticeController.java:77 |
| PUT | `/system/post` | system:post:edit | SysPostController.java:92 |
| PUT | `/system/role` | system:role:edit | SysRoleController.java:110 |
| PUT | `/system/role/authUser/cancel` | system:role:edit | SysRoleController.java:211 |
| PUT | `/system/role/authUser/cancelAll` | system:role:edit | SysRoleController.java:222 |
| PUT | `/system/role/authUser/selectAll` | system:role:edit | SysRoleController.java:233 |
| PUT | `/system/role/changeStatus` | system:role:edit | SysRoleController.java:152 |
| PUT | `/system/role/dataScope` | system:role:edit | SysRoleController.java:139 |
| PUT | `/system/user` | system:user:edit | SysUserController.java:149 |
| PUT | `/system/user/authRole` | system:user:edit | SysUserController.java:236 |
| PUT | `/system/user/changeStatus` | system:user:edit | SysUserController.java:207 |
| PUT | `/system/user/profile` | (仅需登录) | SysProfileController.java:61 |
| PUT | `/system/user/profile/updatePwd` | (仅需登录) | SysProfileController.java:91 |
| PUT | `/system/user/resetPwd` | system:user:resetPwd | SysUserController.java:192 |

## 二期延后（本轮不实现路由）

| Method | Path | 权限串 | Java 来源 |
|---|---|---|---|
| DELETE | `/monitor/job/:jobIds` | monitor:job:remove | SysJobController.java:177 |
| DELETE | `/monitor/jobLog/:jobLogIds` | monitor:job:remove | SysJobLogController.java:73 |
| DELETE | `/monitor/jobLog/clean` | monitor:job:remove | SysJobLogController.java:84 |
| DELETE | `/test/user/:userId` | (仅需登录) | TestController.java:73 |
| DELETE | `/tool/gen/:tableIds` | tool:gen:remove | GenController.java:178 |
| GET | `/captchaImage` | (仅需登录) | CaptchaController.java:31 |
| GET | `/monitor/job/:jobId` | monitor:job:query | SysJobController.java:70 |
| GET | `/monitor/job/list` | monitor:job:list | SysJobController.java:39 |
| GET | `/monitor/jobLog/:jobLogId` | monitor:job:query | SysJobLogController.java:62 |
| GET | `/monitor/jobLog/list` | monitor:job:list | SysJobLogController.java:31 |
| GET | `/test/user/:userId` | (仅需登录) | TestController.java:46 |
| GET | `/test/user/list` | (仅需登录) | TestController.java:38 |
| GET | `/tool/gen/:tableId` | tool:gen:query | GenController.java:70 |
| GET | `/tool/gen/batchGenCode` | tool:gen:code | GenController.java:241 |
| GET | `/tool/gen/column/:tableId` | tool:gen:list | GenController.java:99 |
| GET | `/tool/gen/db/list` | tool:gen:list | GenController.java:87 |
| GET | `/tool/gen/download/:tableName` | tool:gen:code | GenController.java:201 |
| GET | `/tool/gen/genCode/:tableName` | tool:gen:code | GenController.java:213 |
| GET | `/tool/gen/list` | tool:gen:list | GenController.java:49 |
| GET | `/tool/gen/preview/:tableId` | tool:gen:preview | GenController.java:190 |
| GET | `/tool/gen/synchDb/:tableName` | tool:gen:edit | GenController.java:229 |
| GET | `getInfo` | (仅需登录) | SysLoginController.java:72 |
| GET | `getRouters` | (仅需登录) | SysLoginController.java:101 |
| POST | `/login` | (仅需登录) | SysLoginController.java:35 |
| POST | `/monitor/job` | monitor:job:add | SysJobController.java:80 |
| POST | `/monitor/job/export` | monitor:job:export | SysJobController.java:57 |
| POST | `/monitor/jobLog/export` | monitor:job:export | SysJobLogController.java:49 |
| POST | `/system/config/export` | system:config:export | SysConfigController.java:49 |
| POST | `/system/dict/data/export` | system:dict:export | SysDictDataController.java:52 |
| POST | `/system/dict/type/export` | system:dict:export | SysDictTypeController.java:46 |
| POST | `/system/post/export` | system:post:export | SysPostController.java:49 |
| POST | `/system/role/export` | system:role:export | SysRoleController.java:65 |
| POST | `/system/user/export` | system:user:export | SysUserController.java:68 |
| POST | `/system/user/importData` | system:user:import | SysUserController.java:78 |
| POST | `/system/user/importTemplate` | (仅需登录) | SysUserController.java:90 |
| POST | `/test/user/save` | (仅需登录) | TestController.java:61 |
| POST | `/tool/gen/createTable` | (仅需登录) | GenController.java:128 |
| POST | `/tool/gen/importTable` | tool:gen:import | GenController.java:113 |
| PUT | `/monitor/job` | monitor:job:edit | SysJobController.java:116 |
| PUT | `/monitor/job/changeStatus` | monitor:job:changeStatus | SysJobController.java:152 |
| PUT | `/monitor/job/run` | monitor:job:changeStatus | SysJobController.java:165 |
| PUT | `/test/user/update` | (仅需登录) | TestController.java:73 |
| PUT | `/tool/gen` | tool:gen:edit | GenController.java:165 |
