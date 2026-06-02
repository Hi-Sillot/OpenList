package openlistlib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/OpenListTeam/OpenList/v4/cmd"
	"github.com/OpenListTeam/OpenList/v4/cmd/flags"
	"github.com/OpenListTeam/OpenList/v4/internal/bootstrap"
	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/encv"
	"github.com/OpenListTeam/OpenList/v4/openlistlib/internal"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/OpenListTeam/OpenList/v4/server"
	"github.com/OpenListTeam/sftpd-openlist"
	encvPlugins "github.com/Soltus/encv-go/pkg/encv/plugins"
	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var event Event
var logFormatter *internal.MyFormatter

// Init 初始化 openlistlib，包括 cmd 初始化、日志回调注册与退出回调绑定。
// 必须在 Start 之前调用一次。
func Init(e Event, cb LogCallback) error {
	event = e
	cmd.Init()
	logFormatter = &internal.MyFormatter{
		OnLog: func(entry *log.Entry) {
			cb.OnLog(int16(entry.Level), entry.Time.UnixMilli(), entry.Message)
		},
	}
	if utils.Log == nil {
		return errors.New("utils.log is nil")
	} else {
		utils.Log.SetFormatter(logFormatter)
		utils.Log.ExitFunc = event.OnProcessExit
	}
	return nil
}

var httpSrv, httpsSrv, unixSrv *http.Server

func listenAndServe(t string, srv *http.Server) {
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		event.OnStartError(t, err.Error())
	} else {
		event.OnShutdown(t)
	}
}

// IsRunning 判断指定协议（http / https / unix）的服务是否仍在运行。
// t 为空时返回所有服务是否都还在运行。
func IsRunning(t string) bool {
	switch t {
	case "http":
		return httpSrv != nil
	case "https":
		return httpsSrv != nil
	case "unix":
		return unixSrv != nil
	}

	return httpSrv != nil && httpsSrv != nil && unixSrv != nil
}

// Start 启动 OpenList 服务。
// 流程：动态生成 ENCV 配置项 → 加载并初始化 ENCV 插件 →
// 初始化离线下载工具 / 存储 / 任务管理器 → 注册 HTTP 路由并按配置启动监听。
func Start() {
	// 1. 动态生成 ENCV 插件的设置项
	encv.GenerateENCVSettingItems()

	// 2. 从设置中加载 ENCV 插件配置并完成插件初始化
	userSettings, err := encv.LoadENCVPluginSettings()
	if err != nil {
		log.Fatalf("Failed to load ENCV plugin settings: %v", err)
	}
	if err := encvPlugins.InitializeWithSettings(userSettings); err != nil {
		log.Fatalf("Failed to initialize ENCV plugins: %v", err)
	}
	log.Info("ENCV plugins initialized successfully.")

	if conf.Conf.DelayedStart != 0 {
		utils.Log.Infof("delayed start for %d seconds", conf.Conf.DelayedStart)
		time.Sleep(time.Duration(conf.Conf.DelayedStart) * time.Second)
	}
	bootstrap.InitOfflineDownloadTools()
	bootstrap.LoadStorages()
	bootstrap.InitTaskManager()
	if !flags.Debug && !flags.Dev {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.LoggerWithWriter(log.StandardLogger().Out), gin.RecoveryWithWriter(log.StandardLogger().Out))
	server.Init(r)

	if conf.Conf.Scheme.HttpPort != -1 {
		httpBase := fmt.Sprintf("%s:%d", conf.Conf.Scheme.Address, conf.Conf.Scheme.HttpPort)
		utils.Log.Infof("start HTTP server @ %s", httpBase)
		httpSrv = &http.Server{Addr: httpBase, Handler: r}
		go func() {
			listenAndServe("http", httpSrv)
			httpSrv = nil
		}()
	}
	if conf.Conf.Scheme.HttpsPort != -1 {
		httpsBase := fmt.Sprintf("%s:%d", conf.Conf.Scheme.Address, conf.Conf.Scheme.HttpsPort)
		utils.Log.Infof("start HTTPS server @ %s", httpsBase)
		httpsSrv = &http.Server{Addr: httpsBase, Handler: r}
		go func() {
			listenAndServe("https", httpsSrv)
			httpsSrv = nil
		}()
	}
	if conf.Conf.Scheme.UnixFile != "" {
		utils.Log.Infof("start unix server @ %s", conf.Conf.Scheme.UnixFile)
		unixSrv = &http.Server{Handler: r}
		go func() {
			listener, err := net.Listen("unix", conf.Conf.Scheme.UnixFile)
			if err != nil {
				event.OnStartError("unix", err.Error())
			} else {
				mode, err := strconv.ParseUint(conf.Conf.Scheme.UnixFilePerm, 8, 32)
				if err != nil {
					utils.Log.Errorf("failed to parse socket file permission: %+v", err)
				} else {
					err = os.Chmod(conf.Conf.Scheme.UnixFile, os.FileMode(mode))
					if err != nil {
						utils.Log.Errorf("failed to chmod socket file: %+v", err)
					}
				}
				err = unixSrv.Serve(listener)
				if err != nil && err != http.ErrServerClosed {
					event.OnStartError("unix", err.Error())
				}
			}

			unixSrv = nil
		}()
	}
	if conf.Conf.S3.Port != -1 && conf.Conf.S3.Enable {
		s3r := gin.New()
		s3r.Use(gin.LoggerWithWriter(log.StandardLogger().Out), gin.RecoveryWithWriter(log.StandardLogger().Out))
		server.InitS3(s3r)
		s3Base := fmt.Sprintf("%s:%d", conf.Conf.Scheme.Address, conf.Conf.S3.Port)
		fmt.Printf("start S3 server @ %s\n", s3Base)
		utils.Log.Infof("start S3 server @ %s", s3Base)
		go func() {
			var err error
			if conf.Conf.S3.SSL {
				httpsSrv = &http.Server{Addr: s3Base, Handler: s3r}
				err = httpsSrv.ListenAndServeTLS(conf.Conf.Scheme.CertFile, conf.Conf.Scheme.KeyFile)
			}
			if !conf.Conf.S3.SSL {
				httpSrv = &http.Server{Addr: s3Base, Handler: s3r}
				err = httpSrv.ListenAndServe()
			}
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				utils.Log.Fatalf("failed to start s3 server: %s", err.Error())
			}
		}()
	}
	var ftpDriver *server.FtpMainDriver
	var ftpServer *ftpserver.FtpServer
	if conf.Conf.FTP.Listen != "" && conf.Conf.FTP.Enable {
		var err error
		ftpDriver, err = server.NewMainDriver()
		if err != nil {
			utils.Log.Fatalf("failed to start ftp driver: %s", err.Error())
		} else {
			fmt.Printf("start ftp server on %s\n", conf.Conf.FTP.Listen)
			utils.Log.Infof("start ftp server on %s", conf.Conf.FTP.Listen)
			go func() {
				ftpServer = ftpserver.NewFtpServer(ftpDriver)
				err = ftpServer.ListenAndServe()
				if err != nil {
					utils.Log.Fatalf("problem ftp server listening: %s", err.Error())
				}
			}()
		}
	}
	var sftpDriver *server.SftpDriver
	var sftpServer *sftpd.SftpServer
	if conf.Conf.SFTP.Listen != "" && conf.Conf.SFTP.Enable {
		var err error
		sftpDriver, err = server.NewSftpDriver()
		if err != nil {
			utils.Log.Fatalf("failed to start sftp driver: %s", err.Error())
		} else {
			fmt.Printf("start sftp server on %s", conf.Conf.SFTP.Listen)
			utils.Log.Infof("start sftp server on %s", conf.Conf.SFTP.Listen)
			go func() {
				sftpServer = sftpd.NewSftpServer(sftpDriver)
				err = sftpServer.RunServer()
				if err != nil {
					utils.Log.Fatalf("problem sftp server listening: %s", err.Error())
				}
			}()
		}
	}
}

func shutdown(srv *http.Server, timeout time.Duration) error {
	if srv == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := srv.Shutdown(ctx)

	return err
}

// Shutdown 优雅关闭所有 HTTP / HTTPS / UNIX 服务，timeout 单位为毫秒。
// 关闭完成后会自动执行一次数据库 WAL checkpoint。
func Shutdown(timeout int64) (err error) {
	timeoutDuration := time.Duration(timeout) * time.Millisecond
	utils.Log.Println("Shutdown server...")
	if conf.Conf.Scheme.HttpPort != -1 {
		err := shutdown(httpSrv, timeoutDuration)
		if err != nil {
			return err
		}
		httpSrv = nil
		utils.Log.Println("Server HTTP Shutdown")
	}
	if conf.Conf.Scheme.HttpsPort != -1 {
		err := shutdown(httpsSrv, timeoutDuration)
		if err != nil {
			return err
		}
		httpsSrv = nil
		utils.Log.Println("Server HTTPS Shutdown")
	}
	if conf.Conf.Scheme.UnixFile != "" {
		err := shutdown(unixSrv, timeoutDuration)
		if err != nil {
			return err
		}
		unixSrv = nil
		utils.Log.Println("Server UNIX Shutdown")
	}

	ForceDBSync()
	return nil
}

// ForceDBSync 强制 SQLite 执行 WAL checkpoint，将 WAL 中的数据写回主库文件，
// 常用于服务退出前确保数据落盘。
func ForceDBSync() error {
	log.Info("Forcing database sync (WAL checkpoint)...")

	gormDB := db.GetDb()
	if gormDB != nil {
		sqlDB, err := gormDB.DB()
		if err != nil {
			log.Errorf("Failed to get database connection: %v", err)
			return err
		}

		_, err = sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		if err != nil {
			log.Errorf("Failed to execute WAL checkpoint: %v", err)
			return err
		}

		_, err = sqlDB.Exec("PRAGMA synchronous=FULL")
		if err != nil {
			log.Warnf("Failed to set synchronous mode: %v", err)
		}

		log.Info("Database sync completed successfully")
	} else {
		log.Warn("Database instance is nil, skipping sync")
	}

	return nil
}
