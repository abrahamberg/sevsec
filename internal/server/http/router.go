package http

import (
	"net/http"

	"github.com/abrahamberg/devsec/internal/contract"
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

	r.POST("/api/runtime-env", runtimeEnvHandler)

	return r
}

func runtimeEnvHandler(c *gin.Context) {

	var req contract.RuntimeEnvRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, contract.ErrorResponse{
			Error: "invalid request",
		})
		return
	}

	c.JSON(http.StatusOK, contract.RuntimeEnvResponse{
		Env: map[string]string{
			"DEVSEC_PROJECT": req.Project,
			"SOME_ENV":       "some_value",
		},
	})
}
