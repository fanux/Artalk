package handler

import (
	"github.com/ArtalkJS/Artalk/internal/core"
	"github.com/ArtalkJS/Artalk/internal/entity"
	"github.com/ArtalkJS/Artalk/internal/log"
	"github.com/ArtalkJS/Artalk/server/common"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ParamsGet struct {
	PageKey  string `form:"page_key" validate:"required"`
	SiteName string

	Limit  int `form:"limit"`
	Offset int `form:"offset"`

	FlatMode      bool   `form:"flat_mode"`
	SortBy        string `form:"sort_by"`         // date_asc, date_desc, vote
	ViewOnlyAdmin bool   `form:"view_only_admin"` // 只看 admin

	Search string `form:"search"`

	// Message Center
	Type  string `form:"type"` // ["", "all", "mentions", "mine", "pending", "admin_all", "admin_pending"]
	Name  string `form:"name"`
	Email string `form:"email"`

	SiteID  uint
	SiteAll bool

	IsMsgCenter bool
	User        *entity.User
	IsAdminReq  bool
}

type ResponseGet struct {
	Comments    []entity.CookedComment `json:"comments"`
	Total       int64                  `json:"total"`
	TotalRoots  int64                  `json:"total_roots"`
	Page        entity.CookedPage      `json:"page"`
	Unread      []entity.CookedNotify  `json:"unread"`
	UnreadCount int                    `json:"unread_count"`
	ApiVersion  common.ApiVersionData  `json:"api_version"`
	Conf        common.Map             `json:"conf,omitempty"`
}

// @Summary      Comment List
// @Description  Get a list of comments by some conditions
// @Tags         Comment
// @Param        page_key        formData  string  true   "the comment page_key"
// @Param        site_name       formData  string  false  "the site name of your content scope"
// @Param        limit           formData  int     false  "the limit for pagination"
// @Param        offset          formData  int     false  "the offset for pagination"
// @Param        flat_mode       formData  bool    false  "enable flat_mode"
// @Param        sort_by         formData  string  false  "sort by condition"  Enums(date_asc, date_desc, vote)
// @Param        view_only_admin formData  bool    false  "only show comments by admin"
// @Param        search          formData  string  false  "search keywords"
// @Param        type            formData  string  false  "message center show type"  Enums(all, mentions, mine, pending, admin_all, admin_pending)
// @Param        name            formData  string  false  "the username"
// @Param        email           formData  string  false  "the user email"
// @Security     ApiKeyAuth
// @Success      200  {object}   common.JSONResult{data=ResponseGet}
// @Router       /get  [post]
func CommentGet(app *core.App, router fiber.Router) {
	router.Post("/get", func(c *fiber.Ctx) error {
		var p ParamsGet
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		// use site
		common.UseSite(c, &p.SiteName, &p.SiteID, &p.SiteAll)

		// handle params
		UseCfgFrontend(app, &p)

		// find page
		var page entity.Page
		if !p.SiteAll {
			page = app.Dao().FindPage(p.PageKey, p.SiteName)
			if page.IsEmpty() { // if page not found
				page = entity.Page{
					Key:      p.PageKey,
					SiteName: p.SiteName,
				}
			}
		}

		// find user
		var user entity.User
		if p.Name != "" && p.Email != "" {
			user = app.Dao().FindUser(p.Name, p.Email)
			p.User = &user // init params user field
		}

		// check if admin
		if common.CheckIsAdminReq(app, c) {
			p.IsAdminReq = true
		}

		// check if msg center
		if p.Type != "" && p.Name != "" && p.Email != "" {
			p.IsMsgCenter = true
			p.FlatMode = true // 通知中心强制平铺模式
		}

		// prepare the first query
		findScopes := []func(*gorm.DB) *gorm.DB{}
		if !p.FlatMode {
			// nested_mode prepare the root comments as first query result
			findScopes = append(findScopes, RootComments())
		}
		if !p.IsMsgCenter {
			// pinned comments ignore
			findScopes = append(findScopes, func(d *gorm.DB) *gorm.DB {
				return d.Where("is_pinned = ?", false) // 因为置顶是独立的查询，这里就不再查)
			})
		}

		// search function
		if p.Search != "" {
			findScopes = append(findScopes, CommentSearchScope(app, p))
		}

		// get comments for the first query
		var comments []entity.Comment
		GetCommentQuery(app, c, p, p.SiteID, findScopes...).Scopes(Paginate(p.Offset, p.Limit)).Find(&comments)

		// prepend the pinned comments
		prependPinnedComments(app, c, p, &comments)

		// cook
		cookedComments := app.Dao().CookAllComments(comments)

		switch {
		case !p.FlatMode:
			// ==========
			// 层级嵌套模式
			// ==========

			// 获取 comment 子评论
			for _, parent := range cookedComments { // TODO: Read more children, pagination for children comment
				children := app.Dao().FindCommentChildren(parent.ID, SiteIsolationChecker(app, c, p), AllowedCommentChecker(app, c, p))
				cookedComments = append(cookedComments, app.Dao().CookAllComments(children)...)
			}

		case p.FlatMode:
			// ==========
			// 平铺模式
			// ==========

			// find linked comments (被引用的评论，不单独显示)
			for _, comment := range comments {
				if comment.Rid == 0 || entity.ContainsCookedComment(cookedComments, comment.Rid) {
					continue
				}

				rComment := app.Dao().FindComment(comment.Rid, SiteIsolationChecker(app, c, p)) // 查找被回复的评论
				if rComment.IsEmpty() {
					continue
				}

				rCooked := app.Dao().CookComment(&rComment)
				rCooked.Visible = false // 设置为不可见
				cookedComments = append(cookedComments, rCooked)
			}
		}

		// count comments
		total := CountComments(GetCommentQuery(app, c, p, p.SiteID))
		totalRoots := CountComments(GetCommentQuery(app, c, p, p.SiteID, RootComments()))

		// mark all as read in msg center
		if p.IsMsgCenter {
			app.Dao().UserNotifyMarkAllAsRead(p.User.ID)
		}

		// unread notifies
		var unreadNotifies = []entity.CookedNotify{}
		if p.User != nil {
			unreadNotifies = app.Dao().CookAllNotifies(app.Dao().FindUnreadNotifies(p.User.ID))
		}

		// IP region query
		if app.Conf().IPRegion.Enabled {
			ipRegionService, err := core.AppService[*core.IPRegionService](app)
			if err == nil {
				nCookedComments := []entity.CookedComment{}
				for _, c := range cookedComments {
					rawC := app.Dao().FindComment(c.ID)

					c.IPRegion = ipRegionService.Query(rawC.IP)
					nCookedComments = append(nCookedComments, c)
				}
				cookedComments = nCookedComments
			} else {
				log.Error("[IPRegionService] err: ", err)
			}
		}

		resp := ResponseGet{
			Comments:    cookedComments,
			Total:       total,
			TotalRoots:  totalRoots,
			Page:        app.Dao().CookPage(&page),
			Unread:      unreadNotifies,
			UnreadCount: len(unreadNotifies),
			ApiVersion:  common.GetApiVersionDataMap(),
		}

		if p.Offset == 0 {
			resp.Conf = common.GetApiPublicConfDataMap(app, c)
		}

		return common.RespData(c, resp)
	})
}
