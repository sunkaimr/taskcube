package services

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sunkaimr/taskcube/internal/pkg/common"
	. "github.com/sunkaimr/taskcube/internal/services/types"
	"sort"
	"time"
)

type ScriptService struct {
	Script
}

func (c *ScriptService) CheckParameters(ctx *gin.Context) (bool, common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	// 类型校验
	if c.Kind != ScriptKind {
		return false, common.CodeScriptKindErr, fmt.Errorf("unsupport kind: %s, only support: %s", c.Kind, ScriptKind)
	}

	// 脚本类型校验
	switch ScriptType(c.Metadata.Type) {
	case ScriptTypeShell, ScriptTypePython:
	default:
		return false, common.CodeScriptTypeErr, fmt.Errorf("unsupport type: %s, only support: %s,%s", c.Kind, ScriptTypeShell, ScriptTypePython)
	}

	// Name校验（不允许重名）
	f := Script{
		Kind:     ScriptKind,
		Metadata: ScriptMetadata{Name: c.Metadata.Name},
	}
	exist, err := f.Exist(db)
	if err != nil {
		return false, common.CodeServerErr, err
	}

	if exist {
		return false, common.CodeScriptExisted, fmt.Errorf("%s/%s existed", c.Kind, c.Metadata.Name)
	}

	return true, common.CodeOK, nil
}

func (c *ScriptService) CreateScript(ctx *gin.Context) (common.ServiceCode, error) {
	var err error
	log, db := common.ExtractContext(ctx)

	b, code, err := c.CheckParameters(ctx)
	if !b {
		log.Errorf("CheckParameters not pass, %s", err)
		return code, err
	}

	c.Metadata.Version = common.GenerateVersion()
	c.Metadata.CreateAt = time.Now().Format(time.DateTime)
	err = c.Create(db)
	if err != nil {
		log.Errorf("save model.ScriptModel failed, %s", err)
		return code, err
	}

	// 返回最新的script
	res, err := c.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	}

	if len(res) < 1 {
		return common.CodeScriptNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	// 返回创建的脚本
	c.Script = res[0]
	return common.CodeOK, nil
}

func (c *ScriptService) CheckUpdateParameters(ctx *gin.Context) (bool, common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	// 类型校验
	if c.Kind != ScriptKind {
		return false, common.CodeScriptKindErr, fmt.Errorf("unsupport kind: %s, only support: %s", c.Kind, ScriptKind)
	}

	// 脚本类型校验
	switch ScriptType(c.Metadata.Type) {
	case ScriptTypeShell, ScriptTypePython:
	default:
		return false, common.CodeScriptTypeErr, fmt.Errorf("unsupport type: %s, only support: %s,%s", c.Kind, ScriptTypeShell, ScriptTypePython)
	}

	// Name校验
	f := Script{
		Kind:     ScriptKind,
		Metadata: ScriptMetadata{Name: c.Metadata.Name},
	}
	exist, err := f.Exist(db)
	if err != nil {
		return false, common.CodeServerErr, err
	}

	if !exist {
		return false, common.CodeScriptNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	return true, common.CodeOK, nil
}

func (c *ScriptService) UpdateScript(ctx *gin.Context) (common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	b, code, err := c.CheckUpdateParameters(ctx)
	if !b {
		log.Errorf("CheckUpdateParameters not pass, %s", err)
		return code, err
	}

	c.Metadata.Version = common.GenerateVersion()
	c.Metadata.CreateAt = time.Now().Format(time.DateTime)
	err = c.Create(db)
	if err != nil {
		log.Errorf("save model.ScriptModel(%s/%s) failed, %s", c.Metadata.Name, c.Metadata.Version, err)
		return code, err
	}

	res, err := c.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	}

	if len(res) < 1 {
		return common.CodeScriptNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	// 返回更新后的脚本
	c.Script = res[0]

	// 老化掉多余的版本
	f := Script{
		Kind:     ScriptKind,
		Metadata: ScriptMetadata{Name: c.Metadata.Name},
	}
	res, err = f.Get(db)
	if err == nil && len(res) > common.ReservedVersions {
		sort.Sort(ScriptList(res))
		for i := common.ReservedVersions; i < len(res); i++ {
			err = res[i].Delete(db)
			if err != nil {
				log.Errorf("delete model.ScriptModel(%s/%s) failed, %s", c.Metadata.Name, c.Metadata.Version, err)
			}
		}
	}

	return common.CodeOK, nil
}

func (c *ScriptService) QueryScript(ctx *gin.Context, queryMap map[string]string) (any, common.ServiceCode, error) {
	log, db := common.ExtractContext(ctx)

	res, err := common.NewPageList[[]ScriptModel](db).
		QueryPaging(ctx).
		Order("id desc").
		Query(
			common.FilterFuzzyStringMap(queryMap),
		)
	if err != nil {
		err = fmt.Errorf("query models.ScriptModel failed, %s", err)
		log.Error(err)
		return nil, common.CodeServerErr, err
	}

	ret := common.NewPageList[[]Script](db)
	ret.Page = res.Page
	ret.PageSize = res.PageSize
	ret.Total = res.Total
	for i := range res.Items {
		t := &Script{}
		err = json.Unmarshal([]byte(res.Items[i].Data), &t)
		if err != nil {
			log.Errorf("json.Unmarshal %s/%s/%s failed, %s", res.Items[i].Kind, res.Items[i].Name, res.Items[i].Version, err)
			continue
		}

		ret.Items = append(ret.Items, *t)
	}

	sort.Sort(ScriptList(ret.Items))
	return ret, common.CodeOK, nil
}

func (c *ScriptService) DeleteScript(ctx *gin.Context) (common.ServiceCode, error) {
	_, db := common.ExtractContext(ctx)

	if len(c.Metadata.Name) == 0 {
		return common.CodeScriptNameEmpty, fmt.Errorf("script name cannot be empty")
	}

	f := Script{
		Kind:     ScriptKind,
		Metadata: ScriptMetadata{Name: c.Metadata.Name, Version: c.Metadata.Version},
	}
	res, err := f.Get(db)
	if err != nil {
		return common.CodeServerErr, err
	}

	if len(res) == 0 {
		return common.CodeScriptNotExist, fmt.Errorf("%s/%s not exist", c.Metadata.Name, c.Metadata.Version)
	}

	for _, r := range res {
		err = (&r).Delete(db)
		if err != nil {
			err = fmt.Errorf("delete models.ScriptModel(%s/%s) failed, %s", c.Metadata.Name, c.Metadata.Version, err)
			return common.CodeServerErr, err
		}
	}

	return common.CodeOK, nil
}
