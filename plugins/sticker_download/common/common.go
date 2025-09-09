package common

import "io"

type StickerDatas struct {
	Data            io.Reader
	IsConverted     bool
	IsCustomSticker bool
	StickerCount    int
	StickerIndex    int
	StickerSetName  string // 贴纸包的 urlname
	StickerSetTitle string // 贴纸包名称

	WebP int
	WebM int
	TGS  int
}
