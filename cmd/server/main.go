package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	validator "github.com/go-playground/validator/v10"
	"github.com/googlecloudplatform/pi-delivery/gen/index"
	"github.com/googlecloudplatform/pi-delivery/pkg/service"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.ajitem.com/zapdriver"
	"go.uber.org/zap"
)

var digitsLimit int64 = 1000

func init() {
	pflag.Int("digits_limit", 1000, "Max number of digits a client can get in a single request")
	pflag.Int("port", 8080, "Port to run the server on")
}

type getPiRequest struct {
	NumberOfDigits int   `form:"numberOfDigits,default=100" binding:"min=0,digitslimit"`
	Start          int64 `form:"start,default=0" binding:"min=0"`
	Radix          int   `form:"radix,default=10" binding:"oneof=10 16"`
}

type getPiResponse struct {
	// Content is a string representation of Pi digits.
	// ex. "31415926535897932384626433832795028841971693993"
	Content string `json:"content"`
}

func validateGetPiRequest(sl validator.StructLevel) {
	req := sl.Current().Interface().(getPiRequest)

	isStartInRange := func(radix int, start int64) bool {
		if req.Radix == 10 {
			if req.Start >= index.Decimal.TotalDigits() {
				return false
			}
		} else {
			if req.Start >= index.Hexadecimal.TotalDigits() {
				return false
			}
		}
		return true
	}

	if !isStartInRange(req.Radix, req.Start) {
		sl.ReportError(req.Start, "Start", "Start", "startoutofrange", "")
	}
}

func validateDigitsLimit(fl validator.FieldLevel) bool {
	return fl.Field().Int() <= digitsLimit
}

type handlers struct {
	service *service.Service
}

func (h handlers) getPi(c *gin.Context) {
	log := zap.L()
	defer log.Sync()
	var req getPiRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Error("ShouldBind() failed", zap.Error(err))
		c.AbortWithError(http.StatusBadRequest, err)
	}
	set := index.Decimal
	if req.Radix == 16 {
		set = index.Hexadecimal
	}
	res, err := h.service.Get(c.Request.Context(), set, req.Start, int64(req.NumberOfDigits))
	if err != nil {
		log.Error("Get() failed", zap.Error(err))
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}
	c.JSON(http.StatusOK, &getPiResponse{Content: string(res)})
}

func setupRouter(logger *zap.Logger, service *service.Service) *gin.Engine {
	r := gin.New()
	r.Use(cors.New(cors.Config{AllowAllOrigins: true}))
	r.Use(ginzap.Ginzap(logger, time.RFC3339Nano, true))
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterStructValidation(validateGetPiRequest, getPiRequest{})
		v.RegisterValidation("digitslimit", validateDigitsLimit)
	}

	h := &handlers{
		service: service,
	}
	r.GET("/v1/pi", h.getPi)
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})
	return r
}

func main() {
	ctx := context.Background()

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	viper.SetEnvPrefix("pi")
	viper.AutomaticEnv()

	log, err := zapdriver.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zapdriver.NewProduction() failed: %v", err)
		os.Exit(1)
	}
	defer log.Sync()
	zap.ReplaceGlobals(log)

	digitsLimit = viper.GetInt64("digits_limit")
	service, err := service.NewService(ctx, index.BucketName)
	if err != nil {
		log.Fatal("NewService() failed", zap.Error(err))
	}
	defer service.Close()

	r := setupRouter(log, service)
	r.Use(gin.Recovery())
	r.Use(ginzap.RecoveryWithZap(log, true))
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("port")))
	if err != nil {
		log.Fatal("Listen() failed", zap.Error(err))
	}
	if err := r.RunListener(ln); err != nil {
		log.Fatal("RunListener() failed", zap.Error(err))
	}
	log.Warn("Server existing...")
}
