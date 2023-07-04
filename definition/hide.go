package definition

import (
	"encoding/json"
	"time"
)

type MinionHide struct {
	Servername string    `json:"servername"`
	LAN        []string  `json:"lan"`
	VIP        []string  `json:"vip"`
	Edition    string    `json:"edition"`
	Hash       string    `json:"hash"`   // 原始二进制的哈希（不含隐写内容）
	Size       int64     `json:"size"`   // 原始二进制文件大小（不含隐写内容）
	Tags       []string  `json:"tags"`   // 通过链接下载时的一些标记
	Unload     bool      `json:"unload"` // 不加载任何配置，仅在节点注册时生效
	DownloadAt time.Time `json:"download_at"`
}

func (m MinionHide) String() string {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(raw)
}
