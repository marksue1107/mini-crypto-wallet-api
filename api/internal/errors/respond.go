package errors

import (
	"mini-crypto-wallet-api/models"

	"github.com/gin-gonic/gin"
)

// RespondError writes a uniform {error, code, message} JSON error response
// and aborts the context.
//
// Before this, every handler and middleware built its own ad hoc error
// shape: some had "code", some didn't, some added "details", and messages
// ranged from generic strings to raw err.Error() text. A frontend can't
// build one reliable error handler against that. See docs/AUDIT.md S8.
func RespondError(c *gin.Context, status int, code string, err error) {
	c.JSON(status, models.NewErrorResponse(err, code))
	c.Abort()
}
