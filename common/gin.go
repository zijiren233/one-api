package common

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	json "github.com/json-iterator/go"
	"github.com/songquanpeng/one-api/common/ctxkey"
)

func GetRequestBody(c *gin.Context) ([]byte, error) {
	requestBody, _ := c.Get(ctxkey.KeyRequestBody)
	if requestBody != nil {
		return requestBody.([]byte), nil
	}
	defer c.Request.Body.Close()
	if c.Request.ContentLength < 0 {
		return nil, errors.New("request content length is less than 0")
	}
	buf := make([]byte, c.Request.ContentLength)
	_, err := io.ReadFull(c.Request.Body, buf)
	if err != nil {
		return nil, fmt.Errorf("request body read failed: %w", err)
	}
	c.Set(ctxkey.KeyRequestBody, buf)
	return buf, nil
}

func UnmarshalBodyReusable(c *gin.Context, v any) error {
	requestBody, err := GetRequestBody(c)
	if err != nil {
		return err
	}
	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		err = json.Unmarshal(requestBody, &v)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
	} else {
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		err = c.ShouldBind(&v)
	}
	if err != nil {
		return err
	}
	// Reset request body
	return nil
}

func SetEventStreamHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}
