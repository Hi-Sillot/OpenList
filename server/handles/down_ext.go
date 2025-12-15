package handles

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/setting"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/Soltus/encv-go/pkg/encv/openlist"
	"github.com/Soltus/encv-go/pkg/encv/reader"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// handleEncvPreviewFromLink 从 model.Link 解密并预览 ENCV 文件
func handleEncvPreviewFromLink(c *gin.Context, link *model.Link, obj model.Obj) {
	// 【关键】从 OpenList 的设置中获取我们之前配置的解密密码
	password := setting.GetStr("encv_decrypt_password")
	if password == "" {
		log.Errorf("ENCV decrypt password is not set in settings.")
		common.ErrorPage(c, errors.New("Internal Server Error: Decryption key not configured"), 500)
		return
	}

	var decryptReader io.ReadCloser // 【关键】我们只需要最终的 reader
	var err error

	// ====== 【关键修改】直接在 if/else 中处理，不引入中间的工厂抽象 ======
	if link.URL != "" {
		// 情况1：有 URL，使用远程工厂
		log.Debugf("Using remote decrypt factory for URL: %s", link.URL)
		token := setting.GetStr(conf.Token)
		host := fmt.Sprintf("%s:%d", conf.Conf.Scheme.Address, conf.Conf.Scheme.HttpPort)
		if conf.Conf.Scheme.HttpsPort != -1 {
			host = fmt.Sprintf("%s:%d", conf.Conf.Scheme.Address, conf.Conf.Scheme.HttpsPort)
		}
		urlResolver := openlist.NewOpenListURLResolver(host, token, obj.GetPath())

		// 创建工厂，立即使用，然后关闭
		factory, err := reader.NewRemoteDecryptReaderFactory(link.URL, password, link.Header, urlResolver)
		if err != nil {
			log.Errorf("Failed to create remote decrypt reader factory for %s: %v", link.URL, err)
			common.ErrorPage(c, err, 500)
			return
		}
		defer factory.Close() // 尽早 defer

		decryptReader, err = factory.NewDecryptReader()
		if err != nil {
			log.Errorf("Failed to create remote decrypt reader stream for %s: %v", obj.GetName(), err)
			common.ErrorPage(c, err, 500)
			return
		}
	} else {
		// 情况2：没有 URL，使用本地路径
		localPath := obj.GetPath()
		if localPath != "" {
			log.Debugf("Link URL is empty. Attempting to use local factory with path: %s", localPath)

			// 创建工厂，立即使用，然后关闭
			factory, err := reader.NewDecryptReaderFactory(localPath, password)
			if err != nil {
				log.Errorf("Failed to create local decrypt reader factory for %s: %v", localPath, err)
				common.ErrorPage(c, err, 500)
				return
			}
			defer factory.Close() // 尽早 defer

			decryptReader, err = factory.NewDecryptReader()
			if err != nil {
				log.Errorf("Failed to create local decrypt reader stream for %s: %v", obj.GetName(), err)
				common.ErrorPage(c, err, 500)
				return
			}
		} else {
			err = errors.New("both link.URL and obj.GetPath() are empty, cannot create reader")
			log.Errorf(err.Error())
			common.ErrorPage(c, err, 500)
			return
		}
	}

	defer decryptReader.Close()

	// 根据文件扩展名设置正确的 Content-Type
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(obj.GetName()))
	switch ext {
	case ".sccgv":
		contentType = "video/mp4"
	case ".sccgt":
		contentType = "text/plain; charset=utf-8"
	case ".sccgpdf":
		contentType = "application/pdf"
	case ".sccgi":
		contentType = "image/png"
	}
	c.Header("Content-Type", contentType)

	// 将解密后的内容流式传输给客户端
	_, err = utils.CopyWithBuffer(c.Writer, decryptReader)
	if err != nil {
		log.Errorf("Failed to stream decrypted content for %s: %v", obj.GetName(), err)
	}
}
