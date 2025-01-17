package middleware

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ArtalkJS/Artalk/internal/core"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"golang.org/x/exp/slices"
)

func getCorsAllowOrigins(app *core.App) []string {
	allowURLs := []string{}
	allowURLs = append(allowURLs, app.Conf().TrustedDomains...) // 导入配置中的可信域名
	for _, site := range app.Dao().FindAllSitesCooked() {       // 导入数据库中的站点 urls
		allowURLs = append(allowURLs, site.Urls...)
	}

	allowOrigins := []string{}
	for _, u := range allowURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		urlP, err := url.Parse(u)
		if err != nil || urlP.Scheme == "" || urlP.Host == "" {
			continue
		}

		allowOrigins = append(allowOrigins, fmt.Sprintf("%s://%s", urlP.Scheme, urlP.Host))
	}

	return allowOrigins
}

func isOriginAllowed(app *core.App, origin string) bool {
	return slices.Contains(getCorsAllowOrigins(app), origin)
}

func CorsMiddleware(app *core.App) func(*fiber.Ctx) error {
	return cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return isOriginAllowed(app, origin)
		},
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true, // allow cors with cookies
	})
}
