开发：

```
 go run main.go server
```

简单本地打包：

前置条件：前端资产已存在于 `public/dist` 目录中

linux

```
./build-encv-local.sh
```

windows

```
.\build-encv-local.ps1
```

简单测试

```
.\openlist-windows-amd64.exe admin set 123456
```

修改 data/config.json 的端口确保不冲突

在 Openlist Desktop 中使用：

完整退出Openlist Desktop，在任务管理器杀掉所有openlist 进程

将Openlist Desktop安装目录的 openlist.exe 替换为 openlist.exe.back

将编译好的 openlist-windows-amd64.exe 放置在Openlist Desktop安装目录，重命名为 openlist.exe

重启电脑（必须）
