package json

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type SonicJSON struct {
	Data interface{}
}

func (r SonicJSON) Render(w http.ResponseWriter) error {
	jsonBytes, err := jsonAPI.Marshal(r.Data)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonBytes)
	return err
}

func (r SonicJSON) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

var jsonAPI = sonic.Config{
	UseNumber:            true,
	EscapeHTML:           false,
	SortMapKeys:          false,
	CompactMarshaler:     true,
	NoQuoteTextMarshaler: true,
	NoNullSliceOrMap:     true,
}.Froze()

var (
	successResponse       = mustMarshal(Response{Code: 200, Message: "Success"})
	createdResponse       = mustMarshal(Response{Code: 201, Message: "Created"})
	notFoundResponse      = mustMarshal(Response{Code: 404, Message: "Not Found"})
	unauthorizedResponse  = mustMarshal(Response{Code: 401, Message: "Unauthorized"})
	badRequestResponse    = mustMarshal(Response{Code: 400, Message: "Bad Request"})
	forbiddenResponse     = mustMarshal(Response{Code: 403, Message: "Forbidden"})
	internalErrorResponse = mustMarshal(Response{Code: 500, Message: "Internal Server Error"})
)

func mustMarshal(v interface{}) []byte {
	b, _ := jsonAPI.Marshal(v)
	return b
}

func ResponseJSON(c *gin.Context, httpCode int, message string, data interface{}) {
	if data == nil {
		switch httpCode {
		case 200:
			if message == "Success" {
				c.Data(httpCode, "application/json", successResponse)
				return
			}
		case 201:
			if message == "Created" {
				c.Data(httpCode, "application/json", createdResponse)
				return
			}
		case 400:
			if message == "Bad Request" {
				c.Data(httpCode, "application/json", badRequestResponse)
				return
			}
		case 404:
			if message == "Not Found" {
				c.Data(httpCode, "application/json", notFoundResponse)
				return
			}
		case 401:
			if message == "Unauthorized" {
				c.Data(httpCode, "application/json", unauthorizedResponse)
				return
			}
		case 403:
			if message == "Forbidden" {
				c.Data(httpCode, "application/json", forbiddenResponse)
				return
			}
		case 500:
			if message == "Internal Server Error" {
				c.Data(httpCode, "application/json", internalErrorResponse)
				return
			}
		}
	}

	c.Render(httpCode, SonicJSON{Data: Response{
		Code:    httpCode,
		Message: message,
		Data:    data,
	}})
}

func ResponseOK(c *gin.Context, data interface{}) {
	ResponseJSON(c, 200, "Success", data)
}

func ResponseNotFound(c *gin.Context) {
	ResponseJSON(c, 404, "Not Found", nil)
}

func ResponseUnauthorized(c *gin.Context) {
	ResponseJSON(c, 401, "Unauthorized", nil)
}

func ResponseBadRequest(c *gin.Context, message string) {
	if message == "" {
		message = "Bad Request"
	}
	ResponseJSON(c, 400, message, nil)
}

func ResponseForbidden(c *gin.Context) {
	ResponseJSON(c, 403, "Forbidden", nil)
}

func ResponseCreated(c *gin.Context, data interface{}) {
	ResponseJSON(c, 201, "Created", data)
}

func ResponseInternalError(c *gin.Context) {
	ResponseJSON(c, 500, "Internal Server Error", nil)
}
