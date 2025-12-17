package data

import (
	"encoding/json"
	"fmt"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/encv"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

// extendInitialSettings 追加自定义值到指定配置项
func extendInitialSettings(items []model.SettingItem) []model.SettingItem {
	// 注入预览配置
	for i, item := range items {
		switch item.Key {
		case conf.CustomizeHead:
			items[i].Value = CustomizeHeadContent
		case conf.CustomizeBody:
			items[i].Value = CustomizeBodyContent
		case conf.TextTypes:
			items[i].Value += ",sccgt"
		case conf.AudioTypes:
			items[i].Value += ",sccga"
		case conf.VideoTypes:
			items[i].Value += ",sccgv"
		case conf.ImageTypes:
			items[i].Value += ",sccgi"
		case "external_previews":
			// 要追加的内容必须是一个有效的 JSON 对象
			toAdd := `"/.*/": { "VSCode": "vscode://$url" }`
			// 注意：这里我们构造一个完整的对象来让辅助函数解析
			toAddJSON := fmt.Sprintf(`{ %s }`, toAdd)
			mergedValue, err := mergeJSONValues(item.Value, toAddJSON)
			if err != nil {
				utils.Log.Errorf("Failed to merge settings for %s: %v", item.Key, err)
				continue // 如果合并失败，跳过此项，保留原值
			}
			items[i].Value = mergedValue

		case "iframe_previews":
			// 要追加的内容必须是一个有效的 JSON 对象，注意不能有前导逗号
			toAdd := `{
	"sccgpdf": {
		"ENCV PDF": "http://localhost:1808/openlist/sites/pc_dev/_preview/pdf.html?file=$e_url"
	},
	"sccgt": {
		"ENCV Text": "http://localhost:1808/openlist/sites/pc_dev/_preview/text.html?file=$e_url"
	}
}`
			mergedValue, err := mergeJSONValues(item.Value, toAdd)
			if err != nil {
				utils.Log.Errorf("Failed to merge settings for %s: %v", item.Key, err)
				continue // 如果合并失败，跳过此项，保留原值
			}
			items[i].Value = mergedValue
		}
	}

	// 注入解密配置
	items = append(items, model.SettingItem{
		Key:     conf.EncvDecryptPassword,
		Value:   "", // 默认值为空
		Type:    conf.TypeString,
		Group:   model.GLOBAL,  // 分组【设置-全局】
		Flag:    model.PRIVATE, // 标记为私有 (仅管理员可见，使用 int 常量)
		Help:    "Password used to decrypt ENCV container files (.sccg*). Leave empty if not used.",
		Options: "", // 密码类型不需要选项
		Index:   0,  // 可以设置一个值来控制排序，0 或留空
	})

	encvItems := encv.GenerateENCVSettingItems()
	items = append(items, encvItems...)

	return items
}

// mergeJSONValues 将两个 JSON 字符串合并为一个。
// original 是原始的 JSON 字符串，toAdd 是要追加的 JSON 片段（必须是有效的 JSON 对象）。
// 返回合并后的、格式化的 JSON 字符串。
func mergeJSONValues(original, toAdd string) (string, error) {
	var originalMap map[string]interface{}
	// 尝试解析原始值，如果失败则初始化为空 map
	if err := json.Unmarshal([]byte(original), &originalMap); err != nil {
		utils.Log.Warnf("Failed to unmarshal original JSON for merging, initializing as empty map. Error: %v", err)
		originalMap = make(map[string]interface{})
	}

	var toAddMap map[string]interface{}
	// 解析要追加的内容，这里必须是有效的 JSON 对象
	if err := json.Unmarshal([]byte(toAdd), &toAddMap); err != nil {
		return "", fmt.Errorf("invalid JSON provided to add: %w", err)
	}

	// 将新的键值对合并到原始 map 中
	for k, v := range toAddMap {
		originalMap[k] = v
	}

	// 将合并后的 map 重新序列化为格式化的 JSON 字符串
	// 使用 MarshalIndent 以保持与原始代码相似的缩进格式
	mergedBytes, err := json.MarshalIndent(originalMap, "", "\t")
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged JSON: %w", err)
	}

	return string(mergedBytes), nil
}
