package transport

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type AdminServer struct {
	engine       *gin.Engine
	authService  domain.AdminUserService
	userMgmtSvc  domain.AdminUserManagementService
	statsSvc     domain.AdminStatsService
	settingsRepo domain.SettingsRepository
}

func NewAdminServer(
	authService domain.AdminUserService,
	userMgmtSvc domain.AdminUserManagementService,
	statsSvc domain.AdminStatsService,
	settingsRepo domain.SettingsRepository,
	sessionSecret string,
) *AdminServer {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	// Setup sessions
	store := cookie.NewStore([]byte(sessionSecret))
	engine.Use(sessions.Sessions("session", store))

	// Setup template
	engine.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	})
	engine.LoadHTMLGlob("admin/templates/*.html")

	server := &AdminServer{
		engine:       engine,
		authService:  authService,
		userMgmtSvc:  userMgmtSvc,
		statsSvc:     statsSvc,
		settingsRepo: settingsRepo,
	}

	server.setupRoutes()

	return server
}

func (s *AdminServer) setupRoutes() {
	// Health check endpoint (public)
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Public routes
	s.engine.GET("/", s.handleLoginGET)
	s.engine.POST("/login", s.handleLoginPOST)

	// Protected routes
	protected := s.engine.Group("/")
	protected.Use(s.authMiddleware())
	{
		protected.GET("/dashboard", s.handleDashboard)
		protected.GET("/users", s.handleUsers)
		protected.GET("/users/:id", s.handleUserDetail)
		protected.POST("/users/:id/update-permissions", s.handleUpdatePermissions)
		protected.POST("/users/:id/update-limits", s.handleUpdateLimits)
		protected.GET("/settings", s.handleSettings)
		protected.POST("/settings", s.handleSettingsUpdate)
		protected.GET("/stats", s.handleStats)
		protected.POST("/logout", s.handleLogout)
	}
}

func (s *AdminServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("admin_id")

		if userID == nil {
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

		c.Set("admin_id", userID)
		c.Next()
	}
}

func (s *AdminServer) handleLoginGET(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"error": c.Query("error"),
	})
}

func (s *AdminServer) handleLoginPOST(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	admin, err := s.authService.Authenticate(c.Request.Context(), username, password)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/?error=invalid_credentials")
		return
	}

	session := sessions.Default(c)
	session.Set("admin_id", admin.ID)
	session.Set("admin_username", admin.Username)
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
}

func (s *AdminServer) handleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (s *AdminServer) handleDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	today, _ := s.statsSvc.GetDownloadsToday(ctx)
	month, _ := s.statsSvc.GetDownloadsThisMonth(ctx)
	totalUsers, _ := s.statsSvc.GetTotalUsers(ctx)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"downloads_today":    today,
		"downloads_month":    month,
		"total_users":        totalUsers,
		"current_page":       "dashboard",
	})
}

func (s *AdminServer) handleUsers(c *gin.Context) {
	query := c.Query("q")
	limit := 20

	ctx := c.Request.Context()
	var users []*models.User
	var err error

	if query != "" {
		telegramID, parseErr := strconv.ParseInt(query, 10, 64)
		if parseErr == nil {
			user, getErr := s.userMgmtSvc.GetUserByTelegramID(ctx, telegramID)
			if getErr == nil {
				users = []*models.User{user}
			}
		}
	} else {
		users, _, err = s.userMgmtSvc.GetAllUsers(ctx, limit, 0)
	}

	if err != nil {
		users = []*models.User{}
	}

	c.HTML(http.StatusOK, "users.html", gin.H{
		"users":        users,
		"query":        query,
		"current_page": "users",
	})
}

func (s *AdminServer) handleUserDetail(c *gin.Context) {
	// TODO: Implement user detail page
	c.Redirect(http.StatusTemporaryRedirect, "/users")
}

func (s *AdminServer) handleUpdatePermissions(c *gin.Context) {
	// TODO: Implement
	c.Redirect(http.StatusTemporaryRedirect, "/users")
}

func (s *AdminServer) handleUpdateLimits(c *gin.Context) {
	// TODO: Implement
	c.Redirect(http.StatusTemporaryRedirect, "/users")
}

func (s *AdminServer) handleSettings(c *gin.Context) {
	ctx := c.Request.Context()
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		settings = &models.Settings{}
	}

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"settings":     settings,
		"current_page": "settings",
		"success":      c.Query("success"),
	})
}

func (s *AdminServer) handleSettingsUpdate(c *gin.Context) {
	ctx := c.Request.Context()

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/settings")
		return
	}

	dailyLimit, _ := strconv.Atoi(c.PostForm("default_daily_limit"))
	monthlyLimit, _ := strconv.Atoi(c.PostForm("default_monthly_limit"))
	youtubeAllowed := c.PostForm("default_youtube_allowed") == "on"
	tiktokAllowed := c.PostForm("default_tiktok_allowed") == "on"

	settings.DefaultDailyLimit = dailyLimit
	settings.DefaultMonthlyLimit = monthlyLimit
	settings.DefaultYoutubeAllowed = youtubeAllowed
	settings.DefaultTiktokAllowed = tiktokAllowed

	err = s.settingsRepo.Update(ctx, settings)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/settings?error=update_failed")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/settings?success=1")
}

func (s *AdminServer) handleStats(c *gin.Context) {
	ctx := c.Request.Context()

	today, _ := s.statsSvc.GetDownloadsToday(ctx)
	month, _ := s.statsSvc.GetDownloadsThisMonth(ctx)
	topUsers, topCounts, _ := s.statsSvc.GetTopUsers(ctx, 10)

	c.HTML(http.StatusOK, "stats.html", gin.H{
		"downloads_today":  today,
		"downloads_month":  month,
		"top_users":        topUsers,
		"top_counts":       topCounts,
		"current_page":     "stats",
	})
}

func (s *AdminServer) Run(port string) error {
	return s.engine.Run(":" + port)
}
