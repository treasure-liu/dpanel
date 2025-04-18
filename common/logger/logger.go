package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
	"sync"
)

// LogConfig 日志配置
type LogConfig struct {
	Level       string // 日志级别: debug, info, warn, error
	Format      string // 日志格式: text, json
	ToFile      bool   // 是否输出到文件
	FilePath    string // 日志文件路径
	MaxSize     int    // 单个日志文件最大大小(MB)
	MaxBackups  int    // 最大保留的旧日志文件数
	MaxAge      int    // 日志文件保留天数
	Rotate      bool   // 是否启用日志轮转
	RotateDaily bool   // 是否每日轮转
}

// 默认配置
var defaultLogConfig = LogConfig{
	Level:       "info",
	Format:      "text",
	ToFile:      false,
	FilePath:    "/var/log/dpanel/alert.log",
	MaxSize:     100,   // 默认100MB
	MaxBackups:  7,     // 默认保留7个备份
	MaxAge:      30,    // 默认保留30天
	Rotate:      true,  // 默认启用轮转
	RotateDaily: true,  // 默认每日轮转
}

var (
	currentLogFile *os.File
	logMutex       sync.Mutex
	dailyRotate    bool
	lastRotateDate string
)

// Setup 设置日志配置
func Setup() {
	config := loadLogConfig()
	
	// 设置日志级别
	var level slog.Level
	switch strings.ToLower(config.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	
	// 设置日志输出
	var output io.Writer
	if config.ToFile {
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
			slog.Error("创建日志目录失败", "error", err, "path", filepath.Dir(config.FilePath))
		}
		
		// 打开日志文件
		file, err := openLogFile(config.FilePath)
		if err != nil {
			slog.Error("打开日志文件失败", "error", err, "path", config.FilePath)
			output = os.Stdout
		} else {
			// 保存当前日志文件句柄
			currentLogFile = file
			// 日志同时输出到标准输出和文件
			output = io.MultiWriter(os.Stdout, file)
			
			// 设置全局标志
			dailyRotate = config.RotateDaily
			lastRotateDate = time.Now().Format("2006-01-02")
			
			// 启动日志轮转协程
			if config.Rotate {
				go rotateLogRoutine(config)
			}
		}
	} else {
		output = os.Stdout
	}
	
	// 创建日志处理器
	var handler slog.Handler
	if strings.ToLower(config.Format) == "json" {
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{Level: level})
	}
	
	// 设置全局日志处理器
	slog.SetDefault(slog.New(handler))
	
	slog.Info("日志系统已初始化", 
		"level", config.Level, 
		"format", config.Format, 
		"to_file", config.ToFile,
		"file_path", config.FilePath, 
		"rotate", config.Rotate, 
		"rotate_daily", config.RotateDaily,
		"max_size", fmt.Sprintf("%dMB", config.MaxSize),
		"max_backups", config.MaxBackups,
		"max_age", fmt.Sprintf("%d天", config.MaxAge))
}

// 打开日志文件
func openLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// 日志轮转协程
func rotateLogRoutine(config LogConfig) {
	ticker := time.NewTicker(10 * time.Minute) // 每10分钟检查一次
	defer ticker.Stop()
	
	for range ticker.C {
		logMutex.Lock()
		
		// 检查文件大小是否超过限制
		needRotate := false
		
		// 检查是否需要按日期轮转
		if dailyRotate {
			currentDate := time.Now().Format("2006-01-02")
			if currentDate != lastRotateDate {
				needRotate = true
				lastRotateDate = currentDate
			}
		}
		
		// 检查文件大小
		if currentLogFile != nil && !needRotate {
			if stat, err := currentLogFile.Stat(); err == nil {
				if stat.Size() > int64(config.MaxSize*1024*1024) {
					needRotate = true
				}
			}
		}
		
		// 执行日志轮转
		if needRotate && currentLogFile != nil {
			// 关闭当前日志文件
			currentLogFile.Close()
			
			// 轮转日志文件
			rotateLogFile(config.FilePath)
			
			// 清理过期日志
			cleanupOldLogs(config)
			
			// 重新打开日志文件
			newFile, err := openLogFile(config.FilePath)
			if err == nil {
				currentLogFile = newFile
				
				// 重新设置日志输出
				var level slog.Level
				switch strings.ToLower(config.Level) {
				case "debug":
					level = slog.LevelDebug
				case "info":
					level = slog.LevelInfo
				case "warn":
					level = slog.LevelWarn
				case "error":
					level = slog.LevelError
				default:
					level = slog.LevelInfo
				}
				
				var handler slog.Handler
				if strings.ToLower(config.Format) == "json" {
					handler = slog.NewJSONHandler(io.MultiWriter(os.Stdout, newFile), &slog.HandlerOptions{Level: level})
				} else {
					handler = slog.NewTextHandler(io.MultiWriter(os.Stdout, newFile), &slog.HandlerOptions{Level: level})
				}
				
				slog.SetDefault(slog.New(handler))
				slog.Info("日志文件已轮转", "path", config.FilePath)
			} else {
				slog.Error("轮转后重新打开日志文件失败", "error", err, "path", config.FilePath)
			}
		}
		
		logMutex.Unlock()
	}
}

// 轮转日志文件
func rotateLogFile(filePath string) {
	// 获取日期作为后缀
	timestamp := time.Now().Format("20060102-150405")
	
	// 构建轮转后的文件名
	rotatedFilePath := fmt.Sprintf("%s.%s", filePath, timestamp)
	
	// 重命名当前日志文件
	_ = os.Rename(filePath, rotatedFilePath)
}

// 清理过期日志
func cleanupOldLogs(config LogConfig) {
	dirPath := filepath.Dir(config.FilePath)
	baseName := filepath.Base(config.FilePath)
	
	// 读取目录中的所有文件
	files, err := os.ReadDir(dirPath)
	if err != nil {
		slog.Error("读取日志目录失败", "error", err, "path", dirPath)
		return
	}
	
	// 收集所有轮转的日志文件
	var logFiles []string
	for _, file := range files {
		// 只处理非目录文件
		if file.IsDir() {
			continue
		}
		
		// 检查是否是轮转的日志文件
		name := file.Name()
		if strings.HasPrefix(name, baseName+".") {
			logFiles = append(logFiles, filepath.Join(dirPath, name))
		}
	}
	
	// 如果日志文件数超过最大保留数，删除最旧的文件
	if len(logFiles) > config.MaxBackups {
		// 根据文件名排序（文件名包含时间戳，所以可以直接排序）
		for i := 0; i < len(logFiles)-config.MaxBackups; i++ {
			os.Remove(logFiles[i])
			slog.Debug("删除过期日志文件", "path", logFiles[i])
		}
	}
	
	// 删除超过最大保留天数的文件
	if config.MaxAge > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -config.MaxAge)
		
		for _, logFile := range logFiles {
			fileInfo, err := os.Stat(logFile)
			if err != nil {
				continue
			}
			
			if fileInfo.ModTime().Before(cutoffTime) {
				os.Remove(logFile)
				slog.Debug("删除超期日志文件", "path", logFile, "age", fmt.Sprintf("%d天", config.MaxAge))
			}
		}
	}
}

// loadLogConfig 从环境变量加载日志配置
func loadLogConfig() LogConfig {
	config := defaultLogConfig
	
	// 从环境变量读取配置
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Level = level
	}
	
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = format
	}
	
	if toFile := strings.ToLower(os.Getenv("LOG_TO_FILE")); toFile == "true" {
		config.ToFile = true
	}
	
	if filePath := os.Getenv("LOG_FILE_PATH"); filePath != "" {
		config.FilePath = filePath
	}
	
	// 日志轮转配置
	if maxSize := os.Getenv("LOG_MAX_SIZE"); maxSize != "" {
		if size, err := fmt.Sscanf(maxSize, "%d", &config.MaxSize); err != nil || size <= 0 {
			config.MaxSize = 100 // 默认100MB
		}
	}
	
	if maxBackups := os.Getenv("LOG_MAX_BACKUPS"); maxBackups != "" {
		if backups, err := fmt.Sscanf(maxBackups, "%d", &config.MaxBackups); err != nil || backups < 0 {
			config.MaxBackups = 7 // 默认7个备份
		}
	}
	
	if maxAge := os.Getenv("LOG_MAX_AGE"); maxAge != "" {
		if age, err := fmt.Sscanf(maxAge, "%d", &config.MaxAge); err != nil || age < 0 {
			config.MaxAge = 30 // 默认30天
		}
	}
	
	if rotate := os.Getenv("LOG_ROTATE"); rotate != "" {
		config.Rotate = strings.ToLower(rotate) == "true"
	}
	
	if rotateDaily := os.Getenv("LOG_ROTATE_DAILY"); rotateDaily != "" {
		config.RotateDaily = strings.ToLower(rotateDaily) == "true"
	}
	
	return config
} 