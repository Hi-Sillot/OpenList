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
