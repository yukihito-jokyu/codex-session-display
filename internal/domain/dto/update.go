package dto

// UpdateResult はアップデートチェック結果を格納する構造体です。
type UpdateResult struct {
	HasUpdate   bool   `json:"hasUpdate"`
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	ReleaseURL  string `json:"releaseUrl"`
	DownloadURL string `json:"downloadUrl"`
}
