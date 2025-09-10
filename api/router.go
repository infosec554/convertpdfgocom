package api

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "convertpdfgo/api/docs"
	"convertpdfgo/api/handler"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/pkg/middleware"
	"convertpdfgo/service"
)

// @title           Auth API
// @version         1.0
// @description     Authentication and Authorization API
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func New(services service.IServiceManager, log logger.ILogger) *gin.Engine {
	h := handler.New(services, log)
	r := gin.New()

	// Global middlewares
	r.Use(gin.Recovery())
	r.Use(middleware.RateLimiterMiddleware())

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ===================== AUTH =====================
	auth := r.Group("/auth")
	{
		auth.POST("/signup", h.SignUp)
		auth.POST("/login", h.Login)

		// auth required
		authAuth := auth.Group("")
		authAuth.Use(h.AuthRequired)
		{
			authAuth.POST("/logout", h.Logout)
			authAuth.POST("/change-password", h.ChangePassword)
		}

		// social
		auth.POST("/google", h.GoogleAuth)
		auth.POST("/github", h.GithubAuth)
		auth.POST("/facebook", h.FacebookAuth)

		auth.POST("/request-password-reset", h.RequestPasswordReset) // Email yuborish
		auth.POST("/reset-password", h.ResetPassword)                // Token va yangi parol yuborish
	}

	// ===================== ME =====================
	me := r.Group("/me")
	me.Use(h.AuthRequired)
	{
		me.GET("", h.GetMyProfile)
	}

	// ===================== STATS =====================
	stats := r.Group("/stats")
	stats.Use(h.AuthRequired)
	{
		// Eslatma: oldin /stats ichida yana "/stats/user" bor edi -> 404 sabab
		stats.GET("/user", h.GetUserStats)
	}

	// ===================== PUBLIC STATS =====================
	publicStats := r.Group("/public-stats")
	publicStats.Use(h.AuthOptional)
	{
		publicStats.GET("", h.GetPublicStats)
	}

	// ===================== FILES =====================
	// Public upload: token optional (guest allowed)
	filesPub := r.Group("")
	filesPub.Use(h.AuthOptional)
	{
		filesPub.POST("/file", h.UploadFile)
	}

	// User-owned file CRUD: token required
	files := r.Group("/file")
	files.Use(h.AuthRequired)
	{
		files.GET("", h.ListUserFiles)     // GET /file
		files.GET("/:id", h.GetFile)       // GET /file/:id
		files.DELETE("/:id", h.DeleteFile) // DELETE /file/:id
	}

	// api/router.go (mavjud guruhlaringiz ostida)
	jobs := r.Group("/jobs")
	jobs.Use(h.AuthOptional)
	{
		jobs.GET("/:type/:id/download", h.DownloadJobPrimary)
	}

	// Public form submit
	r.POST("/contact", h.ContactCreate)

	// ===================== PDF SERVICES =====================
	// Optional auth: token bo'lsa user_id bog'lanadi, bo'lmasa guest
	pdf := r.Group("/pdf")
	pdf.Use(h.AuthOptional)
	{
		// Merge-1
		pdf.POST("/merge", h.CreateMergeJob)
		pdf.GET("/merge/:id", h.GetMergeJob)
		pdf.POST("/merge/:id/process", h.ProcessMergeJob)

		// Split-2
		pdf.POST("/split", h.CreateSplitJob)
		pdf.GET("/split/:id", h.GetSplitJob)

		// Remove pages-3
		pdf.POST("/remove-pages", h.CreateRemovePagesJob)
		pdf.GET("/remove-pages/:id", h.GetRemovePagesJob)

		// Extract-4
		pdf.POST("/extract", h.CreateExtractJob)
		pdf.GET("/extract/:id", h.GetExtractJob)

		// Compress-5
		pdf.POST("/compress", h.CreateCompressJob)
		pdf.GET("/compress/:id", h.GetCompressJob)

		// Conversions-6
		pdf.POST("/jpg-to-pdf", h.CreateJPGToPDF)
		pdf.GET("/jpg-to-pdf/:id", h.GetJPGToPDFJob)
		//7
		pdf.POST("/pdf-to-jpg", h.CreatePDFToJPG)
		pdf.GET("/pdf-to-jpg/:id", h.GetPDFToJPG)
		//8
		pdf.POST("/pdf-to-word", h.CreatePDFToWordJob)
		pdf.GET("/pdf-to-word/:id", h.GetPDFToWordJob)
		//9
		pdf.POST("/word-to-pdf", h.CreateWordToPDF)
		pdf.GET("/word-to-pdf/:id", h.GetWordToPDFJob)
		//10
		pdf.POST("/excel-to-pdf", h.CreateExcelToPDF)
		pdf.GET("/excel-to-pdf/:id", h.GetExcelToPDFJob)
		//11
		pdf.POST("/ppt-to-pdf", h.CreatePowerPointToPDF)
		pdf.GET("/ppt-to-pdf/:id", h.GetPowerPointToPDFJob)

		// Edit12
		pdf.POST("/rotate", h.CreateRotateJob)
		pdf.GET("/rotate/:id", h.GetRotateJob)
		//13
		pdf.POST("/crop", h.CreateCropJob)
		pdf.GET("/crop/:id", h.GetCropJob)
		//14
		pdf.POST("/add-page-numbers", h.CreateAddPageNumbersJob)
		pdf.GET("/add-page-numbers/:id", h.GetAddPageNumbersJob)
		// Security15
		pdf.POST("/unlock", h.CreateUnlockJob)
		pdf.GET("/unlock/:id", h.GetUnlockJob)
		//16
		pdf.POST("/protect", h.CreateProtectJob)
		pdf.GET("/protect/:id", h.GetProtectJob)

		// Share17
		pdf.POST("/shares", h.CreateSharedLink)
		pdf.GET("/shares/:token", h.GetSharedLink)
	}

	// ===================== ADMIN =====================
	admin := r.Group("/admin")
	admin.Use(h.AuthRequired, h.RoleGuard("admin"))
	{
		// Users: promote/demote/set role
		admin.POST("/users/:id/promote", h.AdminPromoteUser)
		admin.POST("/users/:id/demote", h.AdminDemoteUser)
		admin.POST("/users/:id/role", h.AdminSetUserRole)
		auth.POST("/refresh-token", h.RefreshToken) // public endpoint (cookie yoki body orqali RT)

		// Logs
		admin.GET("/logs/:id", h.GetLogsByJobID)

		// File lifecycle
		admin.GET("/files/pending-deletion", h.AdminListPendingDeletionFiles)
		admin.POST("/files/cleanup", h.CleanupOldFiles)
		admin.GET("/files/deleted-logs", h.AdminDeletedFilesLogs)
		admin.GET("/files", h.AdminListFiles)
		// Jobs overview
		admin.GET("/jobs", h.AdminListJobs)

		//contact us
		admin.GET("/contacts", h.AdminListContacts)
		admin.GET("/contacts/:id", h.AdminGetContact)
		admin.POST("/contacts/:id/read", h.AdminMarkContactRead)
		admin.DELETE("/contacts/:id", h.AdminDeleteContact)

		// (Keyin qo‘shiladiganlar)
		// admin.GET("/health", h.AdminHealth)
		// admin.POST("/users/:id/ban", h.AdminBanUser)
		// admin.POST("/users/:id/unban", h.AdminUnbanUser)
		// admin.DELETE("/files/:id", h.AdminForceDeleteFile)
		// admin.POST("/shares/:token/revoke", h.AdminRevokeShare)
		// admin.GET("/jobs/count", h.AdminJobsCount)
		// admin.GET("/jobs/:id", h.AdminGetJobByID)
	}

	userPreferences := r.Group("/me")
	userPreferences.Use(h.AuthRequired) // JWT tekshiruvi
	{
		// Foydalanuvchi sozlamalarini olish
		userPreferences.GET("/preferences", h.GetUserPreferences)

		// Foydalanuvchi sozlamalarini yangilash
		userPreferences.PATCH("/preferences", h.UpdateUserPreferences)
	}

	return r
}
