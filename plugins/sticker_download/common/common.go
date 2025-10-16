package common

import (
	"archive/zip"
	"io"
)

var StickerSetSuffix          string = ".zip"
var StickerSetConvertedSuffix string = "_converted.zip"

type StickerDatas struct {
	Data            io.Reader
	ZipFile         *zip.ReadCloser

	IsCached        bool
	IsConverted     bool
	IsCustomSticker bool
	IsCompressed    bool

	StickerCount    int
	StickerIndex    int

	StickerSuffix          string
	StickerConvertedSuffix string

	StickerSetName     string // 贴纸包的 urlname
	StickerSetTitle    string // 贴纸包名称
	StickerSetSize     int64
	StickerSetFileName string
	StickerSetHash     string

	WebP int
	WebM int
	TGS  int
}
