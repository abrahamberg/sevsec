package http

import (
	"github.com/abrahamberg/devsec/internal/server/config"
	"github.com/gin-gonic/gin"
)

type Server struct {
	addr string
}

func NewServer(cfg config.Config) *Server {
	return &Server{
		addr: cfg.HTTPAddr,
	}
}

func (s *Server) Run() error {
	r := NewRouter()
	return r.Run(s.addr)
}

func NewRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return r
}
