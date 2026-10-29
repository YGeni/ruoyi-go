// Package gentpl 代码生成的模板，是 ruoyi-generator/src/main/resources/vm 下
// Velocity 模板的 Go text/template 移植。
//
// DIFF(review) —— 移植范围与取舍，必须说清楚：
//
// 1. **只移植 Java 侧 7 个模板**：domain ///
// // mapper.xml / sql。
//
//	   前端那批 vue 模板（index.vue.vm 等）每个 500-660 行，且生成的是
//	   **前端代码**，对一个 Go 后端没有运行期价值，不移植。
//	2. **只支持 tpl_category = crud**：Velocity 模板里到处是 `#if($table.sub)`
//
// 与 `#elseif($table.tree)` 分支，移植它们等于把 sub-.vm
//
//		（76 行）和树表那套 TreeEntity 语义也搬过来。
//		crud 是 RuoYi 的默认类别，也是页面上绝大多数表用的类别；
//		sub / tree 类别这里**按 crud 渲染并在 options 里注明**，不做静默降级。
//	 3. Velocity 的 `$Field.length > 2 && substring(1,2).matches("[A-Z]")`
//	    （用于 xXxx 这种"首字母本就小写、第二字母大写"的字段，如 xId）
//	    在 Go 侧用模板函数 capJavaField 统一处理，语义一致。
package gentpl

import (
	"strings"
	"text/template"
)

// Files 模板文件名 -> 渲染出的文件路径（相对生成根目录）。
//
// 路径规则照抄 VelocityUtils.getFileName。
var Files = []struct {
	Tpl  string // 模板名
	Path string // 输出路径模板（用同样的字段渲染）
}{
	{"domain.java", "java/main/java/{{.PackageNamePath}}/domain/{{.ClassName}}.java"},
	{"mapper.java", "java/main/java/{{.PackageNamePath}}/mapper/{{.ClassName}}Mapper.java"},
	{"service.java", "java/main/java/{{.PackageNamePath}}/service/I{{.ClassName}}Service.java"},
	{"serviceImpl.java", "java/main/java/{{.PackageNamePath}}/service/impl/{{.ClassName}}ServiceImpl.java"},
	{"controller.java", "java/main/java/{{.PackageNamePath}}/controller/{{.ClassName}}Controller.java"},
	{"mapper.xml", "java/main/resources/mapper/{{.ModuleName}}/{{.ClassName}}Mapper.xml"},
	{"sql.sql", "sql/{{.TableName}}.sql"},
}

var raw = map[string]string{
	"domain.java": `package {{.PackageName}}.domain;

{{range .ImportList}}import {{.}};
{{end}}import org.apache.commons.lang3.builder.ToStringBuilder;
import org.apache.commons.lang3.builder.ToStringStyle;
import com.ruoyi.common.annotation.Excel;
import com.ruoyi.common.core.domain.BaseEntity;

/**
 * {{.FunctionName}}对象 {{.TableName}}
 *
 * @author {{.FunctionAuthor}}
 * @date {{.Datetime}}
 */
public class {{.ClassName}} extends BaseEntity
{
    private static final long serialVersionUID = 1L;

{{range .Columns}}{{if not .IsSuperColumn}}    /** {{.ColumnComment}} */
{{if .IsList}}    @Excel(name = "{{.ExcelComment}}"{{if .IsDateType}}, width = 30, dateFormat = "yyyy-MM-dd"{{end}})
{{end}}    private {{.JavaType}} {{.JavaField}};

{{end}}{{end}}
{{range .Columns}}{{if not .IsSuperColumn}}    public void set{{.CapJavaField}}({{.JavaType}} {{.JavaField}})
    {
        this.{{.JavaField}} = {{.JavaField}};
    }

    public {{.JavaType}} get{{.CapJavaField}}()
    {
        return {{.JavaField}};
    }

{{end}}{{end}}
    @Override
    public String toString() {
        return new ToStringBuilder(this,ToStringStyle.MULTI_LINE_STYLE)
{{range .Columns}}{{if not .IsSuperColumn}}            .append("{{.JavaField}}", get{{.CapJavaField}}())
{{end}}{{end}}            .toString();
    }
}
`,

	"mapper.java": `package {{.PackageName}}.mapper;

import java.util.List;
import {{.PackageName}}.domain.{{.ClassName}};

/**
 * {{.FunctionName}}Mapper接口
 *
 * @author {{.FunctionAuthor}}
 * @date {{.Datetime}}
 */
public interface {{.ClassName}}Mapper
{
    /**
     * 查询{{.FunctionName}}
     *
     * @param {{.Pk.JavaField}} {{.FunctionName}}主键
     * @return {{.FunctionName}}
     */
    public {{.ClassName}} select{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaType}} {{.Pk.JavaField}});

    /**
     * 查询{{.FunctionName}}列表
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return {{.FunctionName}}集合
     */
    public List<{{.ClassName}}> select{{.ClassName}}List({{.ClassName}} {{.ClassNameLower}});

    /**
     * 新增{{.FunctionName}}
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return 结果
     */
    public int insert{{.ClassName}}({{.ClassName}} {{.ClassNameLower}});

    /**
     * 修改{{.FunctionName}}
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return 结果
     */
    public int update{{.ClassName}}({{.ClassName}} {{.ClassNameLower}});

    /**
     * 删除{{.FunctionName}}
     *
     * @param {{.Pk.JavaField}} {{.FunctionName}}主键
     * @return 结果
     */
    public int delete{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaType}} {{.Pk.JavaField}});

    /**
     * 批量删除{{.FunctionName}}
     *
     * @param {{.Pk.JavaField}}s 需要删除的数据主键集合
     * @return 结果
     */
    public int delete{{.ClassName}}By{{.Pk.CapJavaField}}s({{.Pk.JavaType}}[] {{.Pk.JavaField}}s);
}
`,

	"service.java": `package {{.PackageName}}.service;

import java.util.List;
import {{.PackageName}}.domain.{{.ClassName}};

/**
 * {{.FunctionName}}Service接口
 *
 * @author {{.FunctionAuthor}}
 * @date {{.Datetime}}
 */
public interface I{{.ClassName}}Service
{
    /**
     * 查询{{.FunctionName}}
     *
     * @param {{.Pk.JavaField}} {{.FunctionName}}主键
     * @return {{.FunctionName}}
     */
    public {{.ClassName}} select{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaType}} {{.Pk.JavaField}});

    /**
     * 查询{{.FunctionName}}列表
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return {{.FunctionName}}集合
     */
    public List<{{.ClassName}}> select{{.ClassName}}List({{.ClassName}} {{.ClassNameLower}});

    /**
     * 新增{{.FunctionName}}
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return 结果
     */
    public int insert{{.ClassName}}({{.ClassName}} {{.ClassNameLower}});

    /**
     * 修改{{.FunctionName}}
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return 结果
     */
    public int update{{.ClassName}}({{.ClassName}} {{.ClassNameLower}});

    /**
     * 批量删除{{.FunctionName}}
     *
     * @param {{.Pk.JavaField}}s 需要删除的{{.FunctionName}}主键集合
     * @return 结果
     */
    public int delete{{.ClassName}}By{{.Pk.CapJavaField}}s({{.Pk.JavaType}}[] {{.Pk.JavaField}}s);

    /**
     * 删除{{.FunctionName}}信息
     *
     * @param {{.Pk.JavaField}} {{.FunctionName}}主键
     * @return 结果
     */
    public int delete{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaType}} {{.Pk.JavaField}});
}
`,

	"serviceImpl.java": `package {{.PackageName}}.service.impl;

import java.util.List;
{{if .HasDateColumn}}import com.ruoyi.common.utils.DateUtils;
{{end}}import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import {{.PackageName}}.mapper.{{.ClassName}}Mapper;
import {{.PackageName}}.domain.{{.ClassName}};
import {{.PackageName}}.service.I{{.ClassName}}Service;

/**
 * {{.FunctionName}}Service业务层处理
 *
 * @author {{.FunctionAuthor}}
 * @date {{.Datetime}}
 */
@Service
public class {{.ClassName}}ServiceImpl implements I{{.ClassName}}Service
{
    @Autowired
    private {{.ClassName}}Mapper {{.ClassNameLower}}Mapper;

    /**
     * 查询{{.FunctionName}}
     *
     * @param {{.Pk.JavaField}} {{.FunctionName}}主键
     * @return {{.FunctionName}}
     */
    @Override
    public {{.ClassName}} select{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaType}} {{.Pk.JavaField}})
    {
        return {{.ClassNameLower}}Mapper.select{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaField}});
    }

    /**
     * 查询{{.FunctionName}}列表
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return {{.FunctionName}}
     */
    @Override
    public List<{{.ClassName}}> select{{.ClassName}}List({{.ClassName}} {{.ClassNameLower}})
    {
        return {{.ClassNameLower}}Mapper.select{{.ClassName}}List({{.ClassNameLower}});
    }

    /**
     * 新增{{.FunctionName}}
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return 结果
     */
    @Override
    public int insert{{.ClassName}}({{.ClassName}} {{.ClassNameLower}})
    {
{{if .HasCreateTime}}        {{.ClassNameLower}}.setCreateTime(DateUtils.getNowDate());
{{end}}        return {{.ClassNameLower}}Mapper.insert{{.ClassName}}({{.ClassNameLower}});
    }

    /**
     * 修改{{.FunctionName}}
     *
     * @param {{.ClassNameLower}} {{.FunctionName}}
     * @return 结果
     */
    @Override
    public int update{{.ClassName}}({{.ClassName}} {{.ClassNameLower}})
    {
{{if .HasUpdateTime}}        {{.ClassNameLower}}.setUpdateTime(DateUtils.getNowDate());
{{end}}        return {{.ClassNameLower}}Mapper.update{{.ClassName}}({{.ClassNameLower}});
    }

    /**
     * 批量删除{{.FunctionName}}
     *
     * @param {{.Pk.JavaField}}s 需要删除的{{.FunctionName}}主键
     * @return 结果
     */
    @Override
    public int delete{{.ClassName}}By{{.Pk.CapJavaField}}s({{.Pk.JavaType}}[] {{.Pk.JavaField}}s)
    {
        return {{.ClassNameLower}}Mapper.delete{{.ClassName}}By{{.Pk.CapJavaField}}s({{.Pk.JavaField}}s);
    }

    /**
     * 删除{{.FunctionName}}信息
     *
     * @param {{.Pk.JavaField}} {{.FunctionName}}主键
     * @return 结果
     */
    @Override
    public int delete{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaType}} {{.Pk.JavaField}})
    {
        return {{.ClassNameLower}}Mapper.delete{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaField}});
    }
}
`,

	"controller.java": `package {{.PackageName}}.controller;

import java.util.List;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import com.ruoyi.common.annotation.Log;
import com.ruoyi.common.core.controller.BaseController;
import com.ruoyi.common.core.domain.AjaxResult;
import com.ruoyi.common.enums.BusinessType;
import {{.PackageName}}.domain.{{.ClassName}};
import {{.PackageName}}.service.I{{.ClassName}}Service;
import com.ruoyi.common.utils.poi.ExcelUtil;
import com.ruoyi.common.core.page.TableDataInfo;

/**
 * {{.FunctionName}}Controller
 *
 * @author {{.FunctionAuthor}}
 * @date {{.Datetime}}
 */
@RestController
@RequestMapping("/{{.ModuleName}}/{{.BusinessName}}")
public class {{.ClassName}}Controller extends BaseController
{
    @Autowired
    private I{{.ClassName}}Service {{.ClassNameLower}}Service;

    /**
     * 查询{{.FunctionName}}列表
     */
    @PreAuthorize("@ss.hasPermi('{{.PermissionPrefix}}:list')")
    @GetMapping("/list")
    public TableDataInfo list({{.ClassName}} {{.ClassNameLower}})
    {
        startPage();
        List<{{.ClassName}}> list = {{.ClassNameLower}}Service.select{{.ClassName}}List({{.ClassNameLower}});
        return getDataTable(list);
    }

    /**
     * 导出{{.FunctionName}}列表
     */
    @PreAuthorize("@ss.hasPermi('{{.PermissionPrefix}}:export')")
    @Log(title = "{{.FunctionName}}", businessType = BusinessType.EXPORT)
    @PostMapping("/export")
    public void export(HttpServletResponse response, {{.ClassName}} {{.ClassNameLower}})
    {
        List<{{.ClassName}}> list = {{.ClassNameLower}}Service.select{{.ClassName}}List({{.ClassNameLower}});
        ExcelUtil<{{.ClassName}}> util = new ExcelUtil<{{.ClassName}}>({{.ClassName}}.class);
        util.exportExcel(response, list, "{{.FunctionName}}数据");
    }

    /**
     * 获取{{.FunctionName}}详细信息
     */
    @PreAuthorize("@ss.hasPermi('{{.PermissionPrefix}}:query')")
    @GetMapping(value = "/{{"{"}}{{.Pk.JavaField}}{{"}"}}")
    public AjaxResult getInfo(@PathVariable("{{.Pk.JavaField}}") {{.Pk.JavaType}} {{.Pk.JavaField}})
    {
        return success({{.ClassNameLower}}Service.select{{.ClassName}}By{{.Pk.CapJavaField}}({{.Pk.JavaField}}));
    }

    /**
     * 新增{{.FunctionName}}
     */
    @PreAuthorize("@ss.hasPermi('{{.PermissionPrefix}}:add')")
    @Log(title = "{{.FunctionName}}", businessType = BusinessType.INSERT)
    @PostMapping
    public AjaxResult add(@RequestBody {{.ClassName}} {{.ClassNameLower}})
    {
        return toAjax({{.ClassNameLower}}Service.insert{{.ClassName}}({{.ClassNameLower}}));
    }

    /**
     * 修改{{.FunctionName}}
     */
    @PreAuthorize("@ss.hasPermi('{{.PermissionPrefix}}:edit')")
    @Log(title = "{{.FunctionName}}", businessType = BusinessType.UPDATE)
    @PutMapping
    public AjaxResult edit(@RequestBody {{.ClassName}} {{.ClassNameLower}})
    {
        return toAjax({{.ClassNameLower}}Service.update{{.ClassName}}({{.ClassNameLower}}));
    }

    /**
     * 删除{{.FunctionName}}
     */
    @PreAuthorize("@ss.hasPermi('{{.PermissionPrefix}}:remove')")
    @Log(title = "{{.FunctionName}}", businessType = BusinessType.DELETE)
    @DeleteMapping("/{{"{"}}{{.Pk.JavaField}}{{"}"}}s")
    public AjaxResult remove(@PathVariable {{.Pk.JavaType}}[] {{.Pk.JavaField}}s)
    {
        return toAjax({{.ClassNameLower}}Service.delete{{.ClassName}}By{{.Pk.CapJavaField}}s({{.Pk.JavaField}}s));
    }
}
`,

	"mapper.xml": `<?xml version="1.0" encoding="UTF-8" ?>
<!DOCTYPE mapper
PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
"http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="{{.PackageName}}.mapper.{{.ClassName}}Mapper">

    <resultMap type="{{.ClassName}}" id="{{.ClassName}}Result">
{{range .Columns}}        <result property="{{.JavaField}}"    column="{{.ColumnName}}"    />
{{end}}    </resultMap>

    <sql id="select{{.ClassName}}Vo">
        select {{.ColumnNameList}} from {{.TableName}}
    </sql>

    <select id="select{{.ClassName}}List" parameterType="{{.ClassName}}" resultMap="{{.ClassName}}Result">
        <include refid="select{{.ClassName}}Vo"/>
        <where>
{{range .QueryColumns}}            <if test="{{.JavaField}} != null {{if .IsStringType}}and {{.JavaField}}.trim() != ''{{end}}"> and {{.ColumnName}} {{.SQLOp}} {{.SQLRight}}</if>
{{end}}        </where>
    </select>

    <select id="select{{.ClassName}}By{{.Pk.CapJavaField}}" parameterType="{{.Pk.JavaType}}" resultMap="{{.ClassName}}Result">
        <include refid="select{{.ClassName}}Vo"/>
        where {{.Pk.ColumnName}} = #{{"{"}}{{.Pk.JavaField}}}
    </select>

    <insert id="insert{{.ClassName}}" parameterType="{{.ClassName}}"{{if .Pk.IsIncrementBool}} useGeneratedKeys="true" keyProperty="{{.Pk.JavaField}}"{{end}}>
        insert into {{.TableName}}
        <trim prefix="(" suffix=")" suffixOverrides=",">
{{range .InsertColumns}}            <if test="{{.JavaField}} != null{{if .IsRequiredString}} and {{.JavaField}} != ''{{end}}">{{.ColumnName}},</if>
{{end}}         </trim>
        <trim prefix="values (" suffix=")" suffixOverrides=",">
{{range .InsertColumns}}            <if test="{{.JavaField}} != null{{if .IsRequiredString}} and {{.JavaField}} != ''{{end}}">#{{"{"}}{{.JavaField}}},</if>
{{end}}         </trim>
    </insert>

    <update id="update{{.ClassName}}" parameterType="{{.ClassName}}">
        update {{.TableName}}
        <trim prefix="SET" suffixOverrides=",">
{{range .UpdateColumns}}            <if test="{{.JavaField}} != null{{if .IsRequiredString}} and {{.JavaField}} != ''{{end}}">{{.ColumnName}} = #{{"{"}}{{.JavaField}},</if>
{{end}}        </trim>
        where {{.Pk.ColumnName}} = #{{"{"}}{{.Pk.JavaField}}
    </update>

    <delete id="delete{{.ClassName}}By{{.Pk.CapJavaField}}" parameterType="{{.Pk.JavaType}}">
        delete from {{.TableName}} where {{.Pk.ColumnName}} = #{{"{"}}{{.Pk.JavaField}}
    </delete>

    <delete id="delete{{.ClassName}}By{{.Pk.CapJavaField}}s" parameterType="String">
        delete from {{.TableName}} where {{.Pk.ColumnName}} in
        <foreach item="{{.Pk.JavaField}}" collection="array" open="(" separator="," close=")">
            #{{"{"}}{{.Pk.JavaField}}
        </foreach>
    </delete>
</mapper>
`,

	"sql.sql": `-- 菜单 SQL
insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}', '{{.ParentMenuId}}', '1', '{{.BusinessName}}', '{{.ModuleName}}/{{.BusinessName}}/index', 1, 0, 'C', '0', '0', '{{.PermissionPrefix}}:list', '#', 'admin', sysdate(), '', null, '{{.FunctionName}}菜单');

-- 按钮父菜单ID
SELECT @parentId := LAST_INSERT_ID();

-- 按钮 SQL
insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}查询', @parentId, '1',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:query',        '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}新增', @parentId, '2',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:add',          '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}修改', @parentId, '3',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:edit',         '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}删除', @parentId, '4',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:remove',       '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}导出', @parentId, '5',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:export',       '#', 'admin', sysdate(), '', null, '');
`,
}

// Parse 解析全部模板。
func Parse() (map[string]*template.Template, error) {
	out := make(map[string]*template.Template, len(raw))
	for name, body := range raw {
		t, err := template.New(name).Funcs(funcs).Parse(body)
		if err != nil {
			return nil, err
		}
		out[name] = t
	}
	return out, nil
}

var funcs = template.FuncMap{
	"lower": strings.ToLower,
}

// ParsePath 输出路径本身也是模板（含 {{.ClassName}} 之类），单独解析。
func ParsePath(path string) (*template.Template, error) {
	return template.New("path").Parse(path)
}
