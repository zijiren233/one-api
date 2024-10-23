package helper

import (
	"fmt"
	"html/template"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/random"
)

func Interface2String(inter interface{}) string {
	switch inter := inter.(type) {
	case string:
		return inter
	case int:
		return fmt.Sprintf("%d", inter)
	case float64:
		return fmt.Sprintf("%f", inter)
	}
	return "Not Implemented"
}

func UnescapeHTML(x string) interface{} {
	return template.HTML(x)
}

func GenRequestID() string {
	return GetTimeString() + random.GetRandomNumberString(8)
}

func GetResponseID(c *gin.Context) string {
	logID := c.GetString(RequestIdKey)
	return fmt.Sprintf("chatcmpl-%s", logID)
}

func AssignOrDefault(value string, defaultValue string) string {
	if len(value) != 0 {
		return value
	}
	return defaultValue
}

func MessageWithRequestId(message string, id string) string {
	return fmt.Sprintf("%s (request id: %s)", message, id)
}

func String2Int(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return num
}
