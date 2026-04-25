package app

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const latestReleaseURL = "https://api.github.com/repos/lrwh/codex-session-manager/releases/latest"

type UpdateResult struct {
	CurrentVersion string
	LatestVersion  string
	AssetName      string
	Updated        bool
}

type githubRelease struct {
	TagName string               `json:"tag_name"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (a *App) Update(currentVersion string) (UpdateResult, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return UpdateResult{}, err
	}

	latestVersion := normalizeVersion(release.TagName)
	result := UpdateResult{
		CurrentVersion: normalizeVersion(currentVersion),
		LatestVersion:  latestVersion,
	}

	if latestVersion == "" {
		return result, errors.New("未获取到有效的 release 版本")
	}
	if result.CurrentVersion == latestVersion {
		return result, nil
	}

	asset, err := selectReleaseAsset(release.Assets)
	if err != nil {
		return result, err
	}
	result.AssetName = asset.Name

	archivePath, err := downloadReleaseAsset(asset)
	if err != nil {
		return result, err
	}
	defer os.Remove(archivePath)

	extractedPath, err := extractBinaryFromArchive(archivePath)
	if err != nil {
		return result, err
	}
	defer os.Remove(extractedPath)

	if err := replaceExecutable(extractedPath); err != nil {
		return result, err
	}

	result.Updated = true
	return result, nil
}

func fetchLatestRelease() (githubRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "csm-updater")

	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return githubRelease{}, errors.New("GitHub Releases 中还没有可用版本")
	}
	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("获取最新版本失败: %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	return release, nil
}

func selectReleaseAsset(assets []githubReleaseAsset) (githubReleaseAsset, error) {
	prefix := fmt.Sprintf("csm-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range assets {
		if !strings.HasPrefix(asset.Name, prefix) {
			continue
		}
		if runtime.GOOS == "windows" && strings.HasSuffix(asset.Name, ".zip") {
			return asset, nil
		}
		if runtime.GOOS != "windows" && strings.HasSuffix(asset.Name, ".tar.gz") {
			return asset, nil
		}
	}
	return githubReleaseAsset{}, fmt.Errorf("未找到当前平台的安装包: %s/%s", runtime.GOOS, runtime.GOARCH)
}

func downloadReleaseAsset(asset githubReleaseAsset) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "csm-updater")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载安装包失败: %s", resp.Status)
	}

	tempFile, err := os.CreateTemp("", "csm-update-*"+filepath.Ext(asset.Name))
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}

func extractBinaryFromArchive(archivePath string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

func extractFromTarGz(archivePath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	expected := "csm"
	if runtime.GOOS == "windows" {
		expected = "csm.exe"
	}

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(header.Name) != expected {
			continue
		}

		out, err := os.CreateTemp("", "csm-bin-*")
		if err != nil {
			return "", err
		}
		defer out.Close()

		if _, err := io.Copy(out, tarReader); err != nil {
			return "", err
		}
		if err := out.Chmod(0o755); err != nil {
			return "", err
		}
		return out.Name(), nil
	}

	return "", errors.New("安装包中未找到可执行文件")
}

func extractFromZip(archivePath string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	expected := "csm.exe"
	for _, file := range reader.File {
		if filepath.Base(file.Name) != expected {
			continue
		}

		src, err := file.Open()
		if err != nil {
			return "", err
		}

		out, err := os.CreateTemp("", "csm-bin-*")
		if err != nil {
			src.Close()
			return "", err
		}

		if _, err := io.Copy(out, src); err != nil {
			src.Close()
			out.Close()
			return "", err
		}
		src.Close()
		out.Close()

		if err := os.Chmod(out.Name(), 0o755); err != nil {
			return "", err
		}
		return out.Name(), nil
	}

	return "", errors.New("zip 包中未找到可执行文件")
}

func replaceExecutable(extractedPath string) error {
	if runtime.GOOS == "windows" {
		return errors.New("Windows 平台暂不支持运行中自替换，请下载最新 zip 手工替换 csm.exe")
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		execPath = filepath.Clean(execPath)
	}

	replacement, err := os.CreateTemp(filepath.Dir(execPath), "csm-update-*")
	if err != nil {
		return err
	}
	replacementPath := replacement.Name()
	replacement.Close()
	defer os.Remove(replacementPath)

	src, err := os.Open(extractedPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(replacementPath, os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	if err := os.Chmod(replacementPath, 0o755); err != nil {
		return err
	}

	return os.Rename(replacementPath, execPath)
}

func normalizeVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}
