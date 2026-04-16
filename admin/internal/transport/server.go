package transport

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

const (
	csrfSessionKey      = "csrf_token"
	failedLoginCountKey = "failed_login_count"
	failedLoginUntilKey = "failed_login_until"
	maxLoginAttempts    = 5
	loginLockDuration   = 10 * time.Minute
	defaultPagination   = 30
)

type AdminServer struct {
	engine        *gin.Engine
	authService   domain.AdminUserService
	userMgmtSvc   domain.AdminUserManagementService
	statsSvc      domain.AdminStatsService
	settingsRepo  domain.SettingsRepository
	baseTemplate  *template.Template
	loginTemplate *template.Template
	funcMap       template.FuncMap
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
	engine.Use(securityHeadersMiddleware())

	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 8,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
	engine.Use(sessions.Sessions("session", store))

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"isTrue": func(v *bool) bool {
			return v != nil && *v
		},
		"isFalse": func(v *bool) bool {
			return v != nil && !*v
		},
	}

	baseTemplate := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles("/app/admin/templates/base.html"))
	loginTemplate := template.Must(template.New("login.html").Funcs(funcMap).ParseFiles("/app/admin/templates/login.html"))

	server := &AdminServer{
		engine:        engine,
		authService:   authService,
		userMgmtSvc:   userMgmtSvc,
		statsSvc:      statsSvc,
		settingsRepo:  settingsRepo,
		baseTemplate:  baseTemplate,
		loginTemplate: loginTemplate,
		funcMap:       funcMap,
	}

	server.setupRoutes()
	return server
}

func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		c.Header("Content-Security-Policy", "default-src 'self' https://cdn.tailwindcss.com https://unpkg.com; script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://unpkg.com; style-src 'self' 'unsafe-inline'; img-src 'self' data:; form-action 'self'; frame-ancestors 'none';")
		c.Next()
	}
}

func (s *AdminServer) generateCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *AdminServer) ensureCSRFToken(c *gin.Context) string {
	session := sessions.Default(c)
	if token, ok := session.Get(csrfSessionKey).(string); ok && token != "" {
		return token
	}
	token, err := s.generateCSRFToken()
	if err != nil {
		return ""
	}
	session.Set(csrfSessionKey, token)
	_ = session.Save()
	return token
}

func (s *AdminServer) verifyCSRF(c *gin.Context) bool {
	session := sessions.Default(c)
	expected, _ := session.Get(csrfSessionKey).(string)
	provided := c.PostForm("csrf_token")
	return expected != "" && expected == provided
}

func (s *AdminServer) renderPage(c *gin.Context, templateName string, data gin.H) {
	data["csrf_token"] = s.ensureCSRFToken(c)

	contentTemplate, err := template.New(templateName).Funcs(s.funcMap).ParseFiles("/app/admin/templates/" + templateName)
	if err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
		return
	}

	var contentBuf bytes.Buffer
	if err := contentTemplate.ExecuteTemplate(&contentBuf, "content", data); err != nil {
		c.String(http.StatusInternalServerError, "Template execution error: %v", err)
		return
	}

	title := "Admin"
	switch templateName {
	case "dashboard.html":
		title = "Dashboard"
	case "users.html":
		title = "Users"
	case "settings.html":
		title = "Settings"
	case "stats.html":
		title = "Statistics"
	}
	data["title"] = title
	data["content"] = template.HTML(contentBuf.String())

	if err := s.baseTemplate.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Base template error: %v", err)
	}
}

func (s *AdminServer) setupRoutes() {
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	s.engine.GET("/", s.handleLoginGET)
	s.engine.POST("/login", s.handleLoginPOST)

	protected := s.engine.Group("/")
	protected.Use(s.authMiddleware())
	{
		protected.GET("/dashboard", s.handleDashboard)
		protected.GET("/users", s.handleUsers)
		protected.POST("/users/:id/update-permissions", s.handleUpdatePermissions)
		protected.POST("/users/:id/update-limits", s.handleUpdateLimits)
		protected.GET("/settings", s.handleSettings)
		protected.POST("/settings", s.handleSettingsUpdate)
		protected.GET("/stats", s.handleStats)
		protected.POST("/change-password", s.handleChangePassword)
		protected.POST("/logout", s.handleLogout)
	}
}

func (s *AdminServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("admin_id")
		if userID == nil {
			c.Redirect(http.StatusSeeOther, "/")
			c.Abort()
			return
		}
		c.Set("admin_id", userID)
		c.Next()
	}
}

func (s *AdminServer) handleLoginGET(c *gin.Context) {
	csrf := s.ensureCSRFToken(c)
	if err := s.loginTemplate.Execute(c.Writer, gin.H{"error": c.Query("error"), "csrf_token": csrf}); err != nil {
		c.String(http.StatusInternalServerError, "Login template error: %v", err)
	}
}

func (s *AdminServer) handleLoginPOST(c *gin.Context) {
	if !s.verifyCSRF(c) {
		c.Redirect(http.StatusSeeOther, "/?error=csrf")
		return
	}

	session := sessions.Default(c)
	if lockUntilStr, ok := session.Get(failedLoginUntilKey).(string); ok && lockUntilStr != "" {
		if lockUntil, err := time.Parse(time.RFC3339, lockUntilStr); err == nil && time.Now().Before(lockUntil) {
			c.Redirect(http.StatusSeeOther, "/?error=rate_limited")
			return
		}
	}

	username := c.PostForm("username")
	password := c.PostForm("password")
	if len(username) < 3 || len(username) > 64 || len(password) < 8 || len(password) > 256 {
		c.Redirect(http.StatusSeeOther, "/?error=invalid_credentials")
		return
	}

	admin, err := s.authService.Authenticate(c.Request.Context(), username, password)
	if err != nil {
		failed := 1
		if current, ok := session.Get(failedLoginCountKey).(int); ok {
			failed = current + 1
		}
		session.Set(failedLoginCountKey, failed)
		if failed >= maxLoginAttempts {
			session.Set(failedLoginUntilKey, time.Now().Add(loginLockDuration).Format(time.RFC3339))
		}
		_ = session.Save()
		c.Redirect(http.StatusSeeOther, "/?error=invalid_credentials")
		return
	}

	session.Clear()
	session.Set("admin_id", admin.ID)
	session.Set("admin_username", admin.Username)
	csrf, _ := s.generateCSRFToken()
	session.Set(csrfSessionKey, csrf)
	_ = session.Save()

	c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (s *AdminServer) handleLogout(c *gin.Context) {
	if !s.verifyCSRF(c) {
		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}
	session := sessions.Default(c)
	session.Clear()
	_ = session.Save()
	c.Redirect(http.StatusSeeOther, "/")
}

func (s *AdminServer) handleDashboard(c *gin.Context) {
	ctx := c.Request.Context()
	today, _ := s.statsSvc.GetDownloadsToday(ctx)
	month, _ := s.statsSvc.GetDownloadsThisMonth(ctx)
	totalUsers, _ := s.statsSvc.GetTotalUsers(ctx)
	avgPerUser := 0.0
	if totalUsers > 0 {
		avgPerUser = float64(month) / float64(totalUsers)
	}

	s.renderPage(c, "dashboard.html", gin.H{
		"downloads_today":  today,
		"downloads_month":  month,
		"total_users":      totalUsers,
		"avg_per_user":     avgPerUser,
		"current_page":     "dashboard",
		"security_posture": "CSRF + rate limit + security headers enabled",
	})
}

func (s *AdminServer) handleUsers(c *gin.Context) {
	ctx := c.Request.Context()
	query := c.Query("q")
	users := []*models.User{}
	total := 0
	var err error

	if query != "" {
		users, err = s.userMgmtSvc.SearchUsers(ctx, query, defaultPagination)
		total = len(users)
	} else {
		users, total, err = s.userMgmtSvc.GetAllUsers(ctx, defaultPagination, 0)
	}
	if err != nil {
		users = []*models.User{}
	}

	s.renderPage(c, "users.html", gin.H{
		"users":        users,
		"query":        query,
		"total":        total,
		"success":      c.Query("success"),
		"error":        c.Query("error"),
		"current_page": "users",
	})
}

func (s *AdminServer) handleUpdatePermissions(c *gin.Context) {
	if !s.verifyCSRF(c) {
		c.Redirect(http.StatusSeeOther, "/users?error=csrf")
		return
	}
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/users?error=invalid_user")
		return
	}

	parseTriState := func(v string) *bool {
		switch v {
		case "allow":
			b := true
			return &b
		case "deny":
			b := false
			return &b
		default:
			return nil
		}
	}

	canYoutube := parseTriState(c.PostForm("can_youtube"))
	canInstagram := parseTriState(c.PostForm("can_instagram"))
	canTiktok := parseTriState(c.PostForm("can_tiktok"))
	if err := s.userMgmtSvc.UpdateUserPermissions(c.Request.Context(), userID, canYoutube, canInstagram, canTiktok); err != nil {
		c.Redirect(http.StatusSeeOther, "/users?error=update_permissions")
		return
	}
	c.Redirect(http.StatusSeeOther, "/users?success=permissions")
}

func (s *AdminServer) handleUpdateLimits(c *gin.Context) {
	if !s.verifyCSRF(c) {
		c.Redirect(http.StatusSeeOther, "/users?error=csrf")
		return
	}
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/users?error=invalid_user")
		return
	}

	parseLimit := func(value string) (*int, error) {
		if value == "" {
			return nil, nil
		}
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 || v > 10000 {
			return nil, domain.ErrUserNotFound
		}
		return &v, nil
	}

	dailyLimit, err := parseLimit(c.PostForm("daily_limit"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/users?error=invalid_limit")
		return
	}
	monthlyLimit, err := parseLimit(c.PostForm("monthly_limit"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/users?error=invalid_limit")
		return
	}

	if err := s.userMgmtSvc.UpdateUserLimits(c.Request.Context(), userID, dailyLimit, monthlyLimit); err != nil {
		c.Redirect(http.StatusSeeOther, "/users?error=update_limits")
		return
	}
	c.Redirect(http.StatusSeeOther, "/users?success=limits")
}

func (s *AdminServer) handleSettings(c *gin.Context) {
	ctx := c.Request.Context()
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		settings = &models.Settings{}
	}
	s.renderPage(c, "settings.html", gin.H{
		"settings":     settings,
		"current_page": "settings",
		"success":      c.Query("success"),
		"error":        c.Query("error"),
	})
}

func (s *AdminServer) handleSettingsUpdate(c *gin.Context) {
	if !s.verifyCSRF(c) {
		c.Redirect(http.StatusSeeOther, "/settings?error=csrf")
		return
	}
	ctx := c.Request.Context()
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/settings")
		return
	}

	dailyLimit, err := strconv.Atoi(c.PostForm("default_daily_limit"))
	if err != nil || dailyLimit < 1 || dailyLimit > 10000 {
		c.Redirect(http.StatusSeeOther, "/settings?error=invalid_daily")
		return
	}
	monthlyLimit, err := strconv.Atoi(c.PostForm("default_monthly_limit"))
	if err != nil || monthlyLimit < 1 || monthlyLimit > 100000 {
		c.Redirect(http.StatusSeeOther, "/settings?error=invalid_monthly")
		return
	}
	youtubeAllowed := c.PostForm("default_youtube_allowed") == "on"
	instagramAllowed := c.PostForm("default_instagram_allowed") == "on"
	tiktokAllowed := c.PostForm("default_tiktok_allowed") == "on"

	settings.DefaultDailyLimit = dailyLimit
	settings.DefaultMonthlyLimit = monthlyLimit
	settings.DefaultYoutubeAllowed = youtubeAllowed
	settings.DefaultInstagramAllowed = instagramAllowed
	settings.DefaultTiktokAllowed = tiktokAllowed

	if err := s.settingsRepo.Update(ctx, settings); err != nil {
		c.Redirect(http.StatusSeeOther, "/settings?error=update_failed")
		return
	}

	c.Redirect(http.StatusSeeOther, "/settings?success=1")
}

func (s *AdminServer) handleStats(c *gin.Context) {
	ctx := c.Request.Context()
	today, _ := s.statsSvc.GetDownloadsToday(ctx)
	month, _ := s.statsSvc.GetDownloadsThisMonth(ctx)
	topUsers, topCounts, _ := s.statsSvc.GetTopUsers(ctx, 10)
	totalUsers, _ := s.statsSvc.GetTotalUsers(ctx)
	bySource, _ := s.statsSvc.GetDownloadsBySource(ctx)

	s.renderPage(c, "stats.html", gin.H{
		"downloads_today":  today,
		"downloads_month":  month,
		"top_users":        topUsers,
		"top_counts":       topCounts,
		"total_users":      totalUsers,
		"downloads_source": bySource,
		"current_page":     "stats",
	})
}

func (s *AdminServer) handleChangePassword(c *gin.Context) {
	if !s.verifyCSRF(c) {
		c.Redirect(http.StatusSeeOther, "/settings?error=csrf")
		return
	}

	adminID, ok := c.Get("admin_id")
	if !ok {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}
	id, ok := adminID.(int64)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	currentPassword := c.PostForm("current_password")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	if newPassword != confirmPassword {
		c.Redirect(http.StatusSeeOther, "/settings?error=password_mismatch")
		return
	}

	if err := s.authService.ChangePassword(c.Request.Context(), id, currentPassword, newPassword); err != nil {
		c.Redirect(http.StatusSeeOther, "/settings?error=password_update")
		return
	}

	c.Redirect(http.StatusSeeOther, "/settings?success=password")
}

func (s *AdminServer) Run(port string) error {
	return s.engine.Run(":" + port)
}
