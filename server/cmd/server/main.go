package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"iforge/iforge/internal/config"
	"iforge/iforge/internal/container"
	"iforge/iforge/internal/database"
	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/handler"
	"iforge/iforge/internal/router"
	"iforge/iforge/internal/service"
	"iforge/iforge/internal/session"

	_ "iforge/iforge/docs"

	fiberswagger "github.com/gofiber/swagger"
)

// @title iForge API
// @version 1.0
// @description iForge API - A modern Git hosting platform
// @host localhost:8081
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// Get home directory - use project directory for development
	homeDir := os.Getenv("IFORGE_HOME")
	if homeDir == "" {
		// Use executable location to determine data directory
		execPath, err := os.Executable()
		if err != nil {
			log.Fatalf("Failed to get executable path: %v", err)
		}
		// data directory is sibling to executable: <exec_dir>/data
		homeDir = filepath.Join(filepath.Dir(execPath), "data")
	}

	// Convert to absolute path to avoid issues with relative paths in git commands
	absHomeDir, err := filepath.Abs(homeDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for home directory: %v", err)
	}
	homeDir = absHomeDir

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Initialize structured logger
	logDir := filepath.Join(homeDir, "logs")
	if err := gitsvc.InitLogger(logDir, gitsvc.LogLevelInfo); err != nil {
		log.Printf("Warning: Failed to initialize logger: %v", err)
	}
	logger := gitsvc.GetLogger()
	logger.Info("Starting iForge server", map[string]interface{}{
		"home_dir": homeDir,
	})

	// Load configuration file
	var cfg *config.Config
	configPath := config.FindConfigFile()
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		logger.Info("配置文件已加载", map[string]interface{}{
			"path": configPath,
		})
	} else {
		// Use default configuration
		cfg = &config.Config{}
		cfg.SetDefaults()
		logger.Info("未找到配置文件，使用默认配置", nil)
	}

	// Environment variable override (highest priority)
	// LoadFromEnv centralizes loading of all env vars (including IFORGE_MYSQL_HOST etc.)
	cfg.LoadFromEnv()

	// Initialize database
	var dbDSN string

	// Build DSN based on driver type
	switch cfg.Database.Driver {
	case "sqlite":
		// SQLite: use database file under homeDir
		dbPath := filepath.Join(homeDir, "iforge.db")
		dbDSN = dbPath
	case "mysql", "postgres":
		var err error
		dbDSN, err = cfg.Database.GetDSN()
		if err != nil {
			log.Fatalf("构建数据库 DSN 失败: %v", err)
		}
	default:
		log.Fatalf("不支持的数据库驱动: %s", cfg.Database.Driver)
	}

	logger.Info("正在连接数据库", map[string]interface{}{
		"driver": cfg.Database.Driver,
	})

	poolConfig := database.PoolConfig{
		MaxIdleConns: cfg.Database.MaxIdleConns(),
		MaxOpenConns: cfg.Database.MaxOpenConns(),
		MaxLifetime:  time.Duration(cfg.Database.ConnMaxLifetimeSeconds) * time.Second,
		MaxIdleTime:  time.Duration(cfg.Database.ConnMaxIdleTimeSeconds) * time.Second,
	}
	if err := database.InitGORMDB(cfg.Database.Driver, dbDSN, poolConfig); err != nil {
		log.Fatal(err)
	}
	defer database.CloseGORMDB()

	// AutoMigrate first: creates all model tables (additive, idempotent).
	// Must run BEFORE SQL migrations so that data-backfill migrations (e.g.
	// 0002_normalize_collaborator_roles) can reference tables that AutoMigrate
	// just created. On a fresh database (e.g. Docker first boot), running
	// RunMigrations first would fail with "no such table".
	if err := database.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	// Run versioned SQL migrations (idempotent, //go:embed based).
	// Handles destructive/transformative changes (column drops, type changes,
	// complex indexes, data backfills) that AutoMigrate cannot express.
	if err := database.RunMigrations(database.GetGORMDB()); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Load or generate session secret
	// Priority: IFORGE_SESSION_SECRET env var > DB stored secret > generated random
	envSecret := os.Getenv("IFORGE_SESSION_SECRET")
	if envSecret != "" {
		session.SetSecret(envSecret)
		logger.Info("Session secret loaded from environment variable", nil)
	} else {
		dbSecret, err := database.LoadOrGenerateSessionSecret()
		if err != nil {
			log.Fatalf("Failed to load or generate session secret: %v", err)
		}
		session.SetSecret(dbSecret)
		logger.Info("Session secret loaded from database", nil)
	}

	// Check initialization status (replaces CreateDefaultAdmin)
	if err := database.EnsureInitialized(); err != nil {
		log.Fatal(err)
	}

	// Setup repositories path
	reposPath := filepath.Join(homeDir, "repositories")
	os.MkdirAll(reposPath, 0755)

	// SSH server configuration (config.yaml -> env vars override)
	sshEnabled := cfg.Server.SSHEnabled
	if env := os.Getenv("IFORGE_SSH_ENABLED"); env != "" {
		sshEnabled = env != "false"
	}
	sshPort := cfg.Server.SSHPort
	if v := os.Getenv("IFORGE_SSH_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			sshPort = p
		}
	}
	// SSH host for clone URL display (default: localhost)
	sshHost := "localhost"
	if v := os.Getenv("IFORGE_SSH_HOST"); v != "" {
		sshHost = v
	}

	logger.Info("SSH configured", map[string]interface{}{
		"enabled": sshEnabled,
		"host":    sshHost,
		"port":    sshPort,
	})

	// HTTP server configuration (config.yaml -> env vars override)
	httpPort := cfg.Server.HTTPPort
	if v := os.Getenv("IFORGE_HTTP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			httpPort = p
		}
	}

	// External URL configuration (for reverse proxy)
	// When using Nginx reverse proxy, set this to the public URL (e.g., http://iforge.example.com)
	// This affects Git clone URLs and other external-facing URLs
	// Priority: env var -> config.yaml -> default
	externalURL := cfg.Server.ExternalURL
	if env := os.Getenv("IFORGE_EXTERNAL_URL"); env != "" {
		externalURL = env
	}
	if externalURL == "" {
		// Default to localhost with port for development
		externalURL = fmt.Sprintf("http://localhost:%d", httpPort)
	}
	logger.Info("External URL configured", map[string]interface{}{
		"external_url": externalURL,
	})

	// Create dependency container
	c := container.NewContainer(reposPath, homeDir, httpPort, sshPort, sshEnabled)
	c.SetExternalURL(externalURL)
	c.SetDB(database.GetGORMDB())
	c.SetDBDriver(cfg.Database.Driver)
	defer c.PluginManager().Stop()
	defer c.MailService().Stop()

	// Wire the singleton audit service so handler.LogAudit can record
	// administrative actions without each handler needing the service injected.
	handler.SetAuditService(c.AuditService())

	// Load plugins
	if err := c.PluginManager().LoadPlugins(); err != nil {
		log.Printf("Warning: Failed to load plugins: %v", err)
	}

	// Initialize event bus (registers webhook subscriber; other subscribers
	// such as audit/notifications can be registered in container.EventBus).
	c.EventBus()

	// Initialize realtime hub (injects into NotificationService for push)
	c.Hub()

	// Initialize CI/CD services
	c.RunnerService()
	c.PipelineService()
	logger.Info("CI/CD services initialized: builtin shell runner + executor worker", nil)

	// Setup router
	app := router.Setup(c)

	// Swagger API documentation
	app.Get("/swagger/*", fiberswagger.HandlerDefault)

	hostKeyPath := filepath.Join(homeDir, "ssh", "host_ed25519")
	if sshEnabled {
		sshServer := service.NewSSHServer(
			c.SSHKeyService(), c.RepoService(), c.GitClient(),
			c.AccountService(), c.EventBus(), c.CollaboratorService(),
			c.DeployKeyService(),
		)
		go func() {
			if err := sshServer.Start(hostKeyPath, sshPort); err != nil {
				logger.Error("SSH server failed", err, nil)
			}
		}()
	}

	// Start server
	logger.Info("iForge server starting", map[string]interface{}{
		"port":        httpPort,
		"ssh_port":    sshPort,
		"ssh_enabled": sshEnabled,
		"database":    cfg.Database.Driver,
	})

	if err := app.Listen(":" + strconv.Itoa(httpPort)); err != nil {
		log.Fatal(err)
	}
}
