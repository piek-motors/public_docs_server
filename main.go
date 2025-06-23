package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FileInfo represents a file or directory in the table of contents
type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDir        bool      `json:"is_dir"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Extension    string    `json:"extension"`
	RelativePath string    `json:"relative_path"`
	Icon         string    `json:"icon"`
}

// DirectoryData represents the data for the table of contents
type DirectoryData struct {
	Path        string     `json:"path"`
	Files       []FileInfo `json:"files"`
	Directories []FileInfo `json:"directories"`
	TotalFiles  int        `json:"total_files"`
	TotalDirs   int        `json:"total_dirs"`
	ScanTime    time.Time  `json:"scan_time"`
}

var (
	scannedPath string
	port        = ":8080"
)

func main() {
	// Set up logging
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Get the directory to scan from command line argument or use current directory
	if len(os.Args) > 1 {
		scannedPath = os.Args[1]
	} else {
		scannedPath = "."
	}

	// Validate the path exists
	if _, err := os.Stat(scannedPath); os.IsNotExist(err) {
		logrus.Fatalf("Directory does not exist: %s", scannedPath)
	}

	// Get absolute path
	absPath, err := filepath.Abs(scannedPath)
	if err != nil {
		logrus.Fatalf("Error getting absolute path: %v", err)
	}
	scannedPath = absPath

	logrus.Infof("Starting directory scanner for: %s", scannedPath)
	logrus.Infof("Server will be available at: http://localhost%s", port)

	// Set up Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Serve static files
	r.Static("/static", "./static")

	// Routes
	r.GET("/", handleIndex)
	r.GET("/api/scan", handleScanAPI)
	r.GET("/api/scan/:path", handleScanPathAPI)

	// Start server
	if err := r.Run(port); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}

func handleIndex(c *gin.Context) {
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error parsing template: %v", err)
		return
	}

	data := gin.H{
		"Title":       "Directory Scanner",
		"ScannedPath": scannedPath,
	}

	c.Header("Content-Type", "text/html")
	tmpl.Execute(c.Writer, data)
}

func handleScanAPI(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = scannedPath
	}

	// Security check: ensure the requested path is within the scanned directory
	if !strings.HasPrefix(path, scannedPath) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	data, err := scanDirectory(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func handleScanPathAPI(c *gin.Context) {
	requestedPath := c.Param("path")
	
	// Decode the path parameter
	decodedPath := strings.ReplaceAll(requestedPath, ":", "/")
	
	// Construct full path
	fullPath := filepath.Join(scannedPath, decodedPath)
	
	// Security check: ensure the requested path is within the scanned directory
	if !strings.HasPrefix(fullPath, scannedPath) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	data, err := scanDirectory(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func scanDirectory(dirPath string) (*DirectoryData, error) {
	// Get absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path: %v", err)
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("error accessing path: %v", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absPath)
	}

	var files []FileInfo
	var directories []FileInfo

	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == absPath {
			return nil
		}

		// Get relative path from scanned root
		relPath, err := filepath.Rel(scannedPath, path)
		if err != nil {
			return err
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		fileInfo := FileInfo{
			Name:         d.Name(),
			Path:         path,
			IsDir:        d.IsDir(),
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			RelativePath: relPath,
		}

		// Get file extension and icon
		if !d.IsDir() {
			fileInfo.Extension = strings.ToLower(filepath.Ext(d.Name()))
			fileInfo.Icon = getFileIcon(fileInfo.Extension)
		} else {
			fileInfo.Icon = "📁"
		}

		if d.IsDir() {
			directories = append(directories, fileInfo)
		} else {
			files = append(files, fileInfo)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %v", err)
	}

	// Sort directories and files
	sort.Slice(directories, func(i, j int) bool {
		return directories[i].Name < directories[j].Name
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Get relative path for display
	relPath, _ := filepath.Rel(scannedPath, absPath)
	if relPath == "." {
		relPath = "Root"
	}

	return &DirectoryData{
		Path:        relPath,
		Files:       files,
		Directories: directories,
		TotalFiles:  len(files),
		TotalDirs:   len(directories),
		ScanTime:    time.Now(),
	}, nil
}

func getFileIcon(ext string) string {
	iconMap := map[string]string{
		".go":     "🔵",
		".js":     "🟡",
		".ts":     "🔵",
		".py":     "🐍",
		".java":   "☕",
		".cpp":    "🔷",
		".c":      "🔷",
		".h":      "🔷",
		".html":   "🌐",
		".css":    "🎨",
		".json":   "📄",
		".xml":    "📄",
		".yaml":   "📄",
		".yml":    "📄",
		".md":     "📝",
		".txt":    "📄",
		".pdf":    "📕",
		".doc":    "📘",
		".docx":   "📘",
		".xls":    "📊",
		".xlsx":   "📊",
		".ppt":    "📽️",
		".pptx":   "📽️",
		".jpg":    "🖼️",
		".jpeg":   "🖼️",
		".png":    "🖼️",
		".gif":    "🖼️",
		".svg":    "🖼️",
		".mp3":    "🎵",
		".mp4":    "🎬",
		".avi":    "🎬",
		".mov":    "🎬",
		".zip":    "📦",
		".tar":    "📦",
		".gz":     "📦",
		".rar":    "📦",
		".exe":    "⚙️",
		".dmg":    "💿",
		".pkg":    "📦",
		".deb":    "📦",
		".rpm":    "📦",
		".sh":     "📜",
		".bat":    "📜",
		".ps1":    "📜",
	}

	if icon, exists := iconMap[ext]; exists {
		return icon
	}
	return "📄"
} 