package convert

import (
	"compress/gzip"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"trbot/plugins/sticker_download/config"

	"golang.org/x/image/webp"
)

// use golang.org/x/image/webp and image/png
func WebPToPNG(webpPath, pngPath string) error {
	// 打开 WebP 文件
	webpFile, err := os.Open(webpPath)
	if err != nil {
		return fmt.Errorf("打开 WebP 文件失败: %w", err)
	}
	defer webpFile.Close()

	// 解码 WebP 图片
	img, err := webp.Decode(webpFile)
	if err != nil {
		return fmt.Errorf("解码 WebP 失败: %w", err)
	}

	// 创建 PNG 文件
	pngFile, err := os.Create(pngPath)
	if err != nil {
		return fmt.Errorf("创建 PNG 文件失败: %w", err)
	}
	defer pngFile.Close()

	// 编码 PNG
	err = png.Encode(pngFile, img)
	if err != nil {
		return fmt.Errorf("编码 PNG 失败: %w", err)
	}

	return nil
}

// use ffmpeg
func WebMToGif(webmPath, gifPath string) error {
	return exec.Command(config.Config.FFmpegPath, "-i", webmPath, gifPath).Run()
}

// use lottie-converter, gifski and compress/gzip
func TGSToGif(tgsPath, gifPath string) error {
	var fps string = "30"
	if config.Config.TGSConvertFPS != 0 {
		fps = fmt.Sprintf("%d", config.Config.TGSConvertFPS)
	}
	// 创建临时目录
	err := os.MkdirAll(config.TempDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(config.TempDir)

	// 若为 .tgs 文件，解压为 .json
	lottiePath := filepath.Join(config.TempDir, "animation.json")
	inFile, err := os.Open(tgsPath)
	if err != nil {
		return fmt.Errorf("failed to open tgs file: %w", err)
	}
	defer inFile.Close()

	gzReader, err := gzip.NewReader(inFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	outFile, err := os.Create(lottiePath)
	if err != nil {
		return fmt.Errorf("failed to create json file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, gzReader); err != nil {
		return fmt.Errorf("failed to decompress tgs: %w", err)
	}

	// 执行 lottie_to_png
	err = exec.Command(config.Config.LottieToPNGPath,
		"--width", "512",
		"--height", "512",
		"--fps", fps,
		"--threads", "1",
		"--output", config.TempDir,
		lottiePath,
	).Run()
	if err != nil {
		return fmt.Errorf("lottie_to_png failed: %w", err)
	}

	// 查找 PNG 文件
	files, err := filepath.Glob(filepath.Join(config.TempDir, "*.png"))
	if err != nil || len(files) == 0 {
		return fmt.Errorf("no PNG files found: %w", err)
	}
	// 排序文件（按名称）
	// 可以用 sort.Strings(files) 如果你需要强制排序

	// 调用 gifski 生成 GIF
	args := append([]string{
		"--quiet",
		"-o", gifPath,
		"--fps", fps,
		"--height", "512",
		"--width", "512",
		"--quality", "90",
	}, files...)

	err = exec.Command(config.Config.GifskiPath, args...).Run()
	if err != nil {
		return fmt.Errorf("gifski failed: %w", err)
	}

	return nil
}

// use ffmpeg
func MP4ToGif(mp4Path, gifPath string) error {
	return exec.Command(config.Config.FFmpegPath, "-i", mp4Path, gifPath).Run()
}

// func MP4ToWebM(MP4Path, webmPath string) error {
// 	return exec.Command(config.Config.FFmpegPath,
// 		"-i", MP4Path,
// 		"-vf", "scale='if(gt(iw,ih),512,-2)':'if(gt(ih,iw),512,-2)',fps=30",
// 		"-c:v", "libvpx-vp9",
// 		"-crf", "40",
// 		"-b:v", "0",
// 		"-an",
// 		"-limit_filesize", "262144",
// 		webmPath,
// 	).Run()
// }
