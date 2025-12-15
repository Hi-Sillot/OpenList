package encv

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/setting"
	encvPlugins "github.com/Soltus/encv-go/pkg/encv/plugins"
	log "github.com/sirupsen/logrus"
)

// mapENCVTypeToOpenList 将 encv-go 的类型映射到 OpenList 的类型
func mapENCVTypeToOpenList(encvType string) string {
	switch encvType {
	case "string":
		return conf.TypeString
	case "bool":
		return conf.TypeBool
	case "number":
		return conf.TypeNumber
	case "select":
		return conf.TypeSelect
	case "text":
		return conf.TypeText
	default:
		return conf.TypeString // 默认为 string
	}
}

// GenerateENCVSettingItems 动态生成所有 ENCV 插件的配置项
// 这个函数返回一个 SettingItem 切片，用于在 extendInitialSettings 中注册
func GenerateENCVSettingItems() []model.SettingItem {
	log.Info("Generating ENCV plugin setting items...")
	metas := encvPlugins.GetPluginMetas()
	var items []model.SettingItem

	for _, meta := range metas {
		for _, field := range meta.SettingFields {
			item := model.SettingItem{
				// Key 格式: encv_<plugin_name>_<field_key>
				Key:     fmt.Sprintf("encv_%s_%s", meta.Name, field.Key),
				Value:   fmt.Sprintf("%v", field.DefaultValue), // 将默认值转为字符串
				Type:    mapENCVTypeToOpenList(field.Type),
				Group:   model.GLOBAL,
				Flag:    model.PRIVATE,
				Help:    field.Help,
				Options: strings.Join(field.Options, ","), // Options 用逗号分隔
				Index:   0,
			}
			items = append(items, item)
			log.Printf("  -> Generated setting item: %s", item.Key)
		}
	}
	log.Info("ENCV plugin setting items generation complete.")
	return items
}

// LoadENCVPluginSettings 从 OpenList 的设置系统中加载所有已注册的 ENCV 插件配置
// 并将其组装成 encv-go InitializeWithSettings 函数所需的格式
func LoadENCVPluginSettings() (map[string]json.RawMessage, error) {
	metas := encvPlugins.GetPluginMetas()
	allSettings := make(map[string]json.RawMessage)

	for _, meta := range metas {
		pluginConfigMap := make(map[string]interface{}) // 用于存储单个插件的配置

		for _, field := range meta.SettingFields {
			key := fmt.Sprintf("encv_%s_%s", meta.Name, field.Key)

			// 根据类型调用不同的 getter
			var value interface{}
			switch field.Type {
			case "string", "text", "select":
				value = setting.GetStr(key)
			case "bool":
				value = setting.GetBool(key)
			case "number":
				value = setting.GetInt(key, 0) // GetInt 需要一个默认值，我们给 0
			}

			pluginConfigMap[field.Key] = value
		}

		// 将单个插件的配置 map 序列化为 JSON
		jsonData, err := json.Marshal(pluginConfigMap)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal settings for plugin '%s': %w", meta.Name, err)
		}
		allSettings[meta.Name] = jsonData
	}

	return allSettings, nil
}
