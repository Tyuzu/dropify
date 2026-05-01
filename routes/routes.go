package routes

import (
	"dropify/droping"
	"dropify/mediaproxy"
	"dropify/middleware"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func AddStaticRoutes(router *httprouter.Router) {
	// mediaproxy.InitMediaProxy()
	router.ServeFiles("/static/uploads/*filepath", http.Dir("static/uploads"))

	router.GET("/static/proxy/*url", mediaproxy.ProxyHandler)
	// router.GET("/external/:hash/*rest", mediaproxy.ProxyHandler)

}

func AddFiledropRoutes(router *httprouter.Router, rateLimiter *middleware.RateLimiter) {
	// router.GET("/health", droping.HealthHandler)
	router.POST("/api/v1/filedrop", droping.FiledropHandler)
	router.OPTIONS("/api/v1/filedrop", droping.OptionsHandler)
}
