package routes

import (
	"dropify/chunkedup"
	"dropify/droping"
	"dropify/filedrop"
	"dropify/filemgr"
	"dropify/mediaproxy"
	"dropify/middleware"
	"dropify/posts"
	"dropify/ratelim"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func AddStaticRoutes(router *httprouter.Router) {
	// mediaproxy.InitMediaProxy()
	router.ServeFiles("/static/uploads/*filepath", http.Dir("static/uploads"))

	router.GET("/static/proxy/*url", mediaproxy.ProxyHandler)
	// router.GET("/external/:hash/*rest", mediaproxy.ProxyHandler)

}

func AddFiledropRoutes(router *httprouter.Router, rateLimiter *ratelim.RateLimiter) {
	// router.GET("/health", droping.HealthHandler)

	router.POST("/api/v1/filedrop/uploads/chunk", rateLimiter.Limit(chunkedup.ChunkedUploads))
	router.HEAD("/api/v1/filedrop/uploads/exists", chunkedup.FileExistsHandler)

	router.POST("/api/v1/filedrop/chat", rateLimiter.Limit(filedrop.UploadHandler))
	router.POST("/api/v1/filedrop/feedpost", rateLimiter.Limit(posts.UploadImage))

	router.POST("/api/v1/filedrop", droping.FiledropHandler)
	router.OPTIONS("/api/v1/filedrop", droping.OptionsHandler)

	router.PUT("/api/v1/profile/avatar", rateLimiter.Limit(middleware.Authenticate(filedrop.EditProfilePic)))
	router.PUT("/api/v1/banner/:entitytype/:entityid", rateLimiter.Limit(middleware.Authenticate(filemgr.EditBanner)))

	router.PUT("/api/v1/gallery/:entityType/:entityId/images", rateLimiter.Limit(middleware.Authenticate(filedrop.UpdateGalleryImages)))
}
