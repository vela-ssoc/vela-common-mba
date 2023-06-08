package steganography

import "time"

type MinionHide struct {
	Servername string    `json:"servername"`
	LAN        []string  `json:"lan"`
	VIP        []string  `json:"vip"`
	Edition    string    `json:"edition"`
	Hash       string    `json:"hash"`
	Size       int64     `json:"size"`
	DownloadAt time.Time `json:"download_at"`
}