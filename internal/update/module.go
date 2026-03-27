package update

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/module"
)

// DownloadModule 下载模块
func DownloadModule(url string, dest string) error {
	logger.Infof("下载模块：%s", url)

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("创建目录失败：%v", err)
	}

	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败：%v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败：状态码：%d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("创建文件失败：%v", err)
	}
	defer file.Close()

	// 写入文件
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败：%v", err)
	}

	logger.Infof("下载完成：%s", dest)
	return nil
}

// ExtractModule 解压模块
func ExtractModule(src string, dest string) error {
	logger.Infof("解压模块：%s", src)

	// 打开 ZIP 文件
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("打开 ZIP 文件失败：%v", err)
	}
	defer zipReader.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败：%v", err)
	}

	// 遍历 ZIP 文件中的所有文件
	for _, file := range zipReader.File {
		// 构建目标文件路径
		filePath := filepath.Join(dest, file.Name)

		// 检查路径是否合法（防止目录穿越攻击）
		if !strings.HasPrefix(filePath, filepath.Clean(dest)) {
			logger.Warnf("跳过非法路径的文件：%s", file.Name)
			continue
		}

		// 如果是目录，创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("创建目录失败：%v", err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("创建父目录失败：%v", err)
		}

		// 打开 ZIP 中的文件
		zipFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开 ZIP 中的文件失败：%v", err)
		}

		// 创建目标文件
		targetFile, err := os.Create(filePath)
		if err != nil {
			zipFile.Close()
			return fmt.Errorf("创建目标文件失败：%v", err)
		}

		// 复制文件内容
		_, err = io.Copy(targetFile, zipFile)
		zipFile.Close()
		targetFile.Close()

		if err != nil {
			return fmt.Errorf("复制文件内容失败：%v", err)
		}

		// 设置文件权限
		if err := os.Chmod(filePath, file.Mode()); err != nil {
			logger.Warnf("设置文件权限失败：%v", err)
		}
	}

	logger.Infof("解压完成：%s", dest)
	return nil
}

// CreateModule 创建模块实例（热替换时由 ApplyUpdate 调用）
// 注意：仅用于热更新流程（下载 ZIP → 解压 → 创建新实例 → 替换旧实例）
// 如需真实热替换，需将编译好的新二进制加载进来；
// 当前为"原地重建"策略（重新 new 一个相同类型的模块实例）。
func CreateModule(moduleName string, version string) (module.Module, error) {
	switch moduleName {
	case "bilibili":
		return module.NewBilibiliModule(), nil
	case "acfun":
		return module.NewAcfunModule(), nil
	case "youtube":
		return module.NewYoutubeModule(), nil
	case "douyu":
		return module.NewDouyuModule(), nil
	case "huya":
		return module.NewHuyaModule(), nil
	case "weibo":
		return module.NewWeiboModule(), nil
	case "douyin":
		return module.NewDouyinModule(), nil
	case "twitter":
		return module.NewTwitterModule(), nil
	case "twitcasting":
		return module.NewTwitcastingModule(), nil
	default:
		return nil, fmt.Errorf("未知模块：%s", moduleName)
	}
}
