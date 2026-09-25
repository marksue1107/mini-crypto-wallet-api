package models

// PaginationRequest 分頁請求參數
type PaginationRequest struct {
	Page     int `form:"page" json:"page" example:"1"`         // 頁碼，從 1 開始
	PageSize int `form:"page_size" json:"page_size" example:"20"` // 每頁數量
}

// PaginationResponse 分頁響應
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// GetOffset 計算偏移量
func (p *PaginationRequest) GetOffset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit 獲取限制數量
func (p *PaginationRequest) GetLimit() int {
	if p.PageSize < 1 {
		return 20
	}
	if p.PageSize > 100 {
		return 100
	}
	return p.PageSize
}
