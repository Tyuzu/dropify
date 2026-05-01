package routes

import (
	"dropify/infra"
	"dropify/middleware"

	"github.com/julienschmidt/httprouter"
)

func RoutesWrapper(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	AddFiledropRoutes(router, rateLimiter)
}
