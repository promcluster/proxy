package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func (s *Service) auth(c *gin.Context) {
	confUser := strings.TrimSpace(viper.GetString("auth.user"))
	confToken := strings.TrimSpace(viper.GetString("auth.token"))

	if confToken == "" {
		c.AbortWithStatus(401)
		return
	}

	token := strings.TrimSpace(c.Request.Header.Get("Authorization"))

	// Bearer Token support
	if token == fmt.Sprintf("Bearer %s", confToken) {
		c.Next()
		return
	}

	// Basic Auth support
	if u, p, ok := c.Request.BasicAuth(); ok && confToken == p && confUser == u {
		c.Next()
		return
	}

	c.AbortWithStatus(401)
}

func (s *Service) accessLog() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}
