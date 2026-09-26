package models

// ErrorResponse 統一的錯誤響應格式
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewErrorResponse 創建錯誤響應
func NewErrorResponse(err error, code string) ErrorResponse {
	return ErrorResponse{
		Error:   err.Error(),
		Code:    code,
		Message: err.Error(),
	}
}
