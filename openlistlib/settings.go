package openlistlib

import (
	"github.com/OpenListTeam/OpenList/v4/cmd"
	"github.com/OpenListTeam/OpenList/v4/cmd/flags"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

// SetConfigData 设置数据目录（对应 --data）。
func SetConfigData(path string) {
	flags.DataDir = path
}

// SetConfigLogStd 设置是否强制输出日志到 stdout（对应 --log-std）。
func SetConfigLogStd(b bool) {
	flags.LogStd = b
}

// SetConfigDebug 设置是否启用 debug 模式（对应 --debug）。
func SetConfigDebug(b bool) {
	flags.Debug = b
}

// SetConfigNoPrefix 设置是否禁用环境变量前缀（对应 --no-prefix）。
func SetConfigNoPrefix(b bool) {
	flags.NoPrefix = b
}

// SetAdminPassword 重置 admin 用户的密码并清除其登录缓存。
func SetAdminPassword(pwd string) {
	admin, err := op.GetAdmin()
	if err != nil {
		utils.Log.Errorf("failed get admin user: %+v", err)
		return
	}
	admin.SetPassword(pwd)
	if err := op.UpdateUser(admin); err != nil {
		utils.Log.Errorf("failed update admin user: %+v", err)
		return
	}
	utils.Log.Infof("admin user has been updated:")
	utils.Log.Infof("username: %s", admin.Username)
	utils.Log.Infof("password: %s", pwd)
	cmd.DelAdminCacheOnline()
}
