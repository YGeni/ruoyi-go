// Package scheduler 定时任务调度：加载 sys_job，按 cron 触发，执行后写 sys_job_log。
//
// DIFF(review) —— 这是整个迁移里语义偏差最大的一块，必须说清楚：
//
// Java 侧是 **quartz + Spring bean 反射**：invoke_target 形如
// `ryTask.ryParams('ry')`，ScheduleUtils 用 SpringUtils.getBean("ryTask") 取到
// com.ruoyi.quartz.task.RyTask 实例，再反射调它的 ryParams 方法。
// 白名单是 Constants.JOB_WHITELIST_STR = {"com.ruoyi.quartz.task"}——
// 也就是说**只有这一个包下的 bean 能被调**，本质是防 RCE。
//
// Go 侧没有 Spring 容器也没有这种反射。做法是：
//  1. **任务注册表**（本文件）：把白名单里那几个 bean 方法在 Go 里实现一遍，
//     key 就是 `ryTask.xxx`。RuoYi 自带的三个种子任务正好全覆盖。
//  2. 不在注册表里的 invoke_target：新增/修改时按白名单规则拒绝（与 Java 一致，
//     报"目标字符串不在白名单内"）；真到执行时返回"任务不存在或已过期！"
//     —— 与 SysJobServiceImpl.run 找不到任务时的返回一致。
package scheduler

import (
	"fmt"
	"strings"
)

// TaskFunc 一次任务执行。返回 (消息, 错误)；错误非 nil 时日志 status 记为 '1' 失败。
type TaskFunc func(args []string) (string, error)

// registry 任务注册表：key 是 invoke_target 里 `bean.method` 的部分。
var registry = map[string]TaskFunc{
	"ryTask.ryNoParams": func(_ []string) (string, error) {
		return "执行无参方法", nil
	},
	"ryTask.ryParams": func(args []string) (string, error) {
		if len(args) == 0 {
			return "", fmt.Errorf("ryParams 需要一个参数")
		}
		return "执行有参方法：" + args[0], nil
	},
	"ryTask.ryMultipleParams": func(args []string) (string, error) {
		// 打印的是那五个形参；Go 侧按原样拼回字符串。
		return fmt.Sprintf("执行多参方法： 字符串类型%s，布尔类型%s，长整型%s，浮点型%s，整形%s",
			at(args, 0), at(args, 1), at(args, 2), at(args, 3), at(args, 4)), nil
	},
}

func at(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

// Registered 该 bean.method 是否在注册表里（即可执行）。
func Registered(beanMethod string) bool {
	_, ok := registry[beanMethod]
	return ok
}

// parseInvokeTarget 把 `ryTask.ryParams('ry', true, 2000L, 316.50D, 100)` 拆成
// bean.method 与参数列表（去掉引号与类型后缀）。
//
// Java 侧是 JobInvokeUtil 用正则 + 反射做类型推断；Go 侧只需字符串，
// 这里只做拆分，参数一律当字符串传进 TaskFunc。
func parseInvokeTarget(target string) (string, []string, error) {
	target = strings.TrimSpace(target)
	lp := strings.Index(target, "(")
	if lp <= 0 || !strings.HasSuffix(target, ")") {
		return "", nil, fmt.Errorf("调用目标格式不正确")
	}
	beanMethod := strings.TrimSpace(target[:lp])
	inner := strings.TrimSpace(target[lp+1 : len(target)-1])
	if inner == "" {
		return beanMethod, nil, nil
	}
	var args []string
	for _, p := range strings.Split(inner, ",") {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "'\"")
		// 去掉 Java 字面量后缀：2000L / 316.50D / 1F
		p = strings.TrimRight(p, "LlDdFf")
		args = append(args, p)
	}
	return beanMethod, args, nil
}

// Invoke 执行一次任务。
func Invoke(target string) (string, error) {
	beanMethod, args, err := parseInvokeTarget(target)
	if err != nil {
		return "", err
	}
	fn, ok := registry[beanMethod]
	if !ok {
		return "", fmt.Errorf("任务不存在或已过期！")
	}
	return fn(args)
}
