package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
	CanView      bool      `json:"can_view"`
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

type Server struct {
	scannedPath string
	port        string
}

func NewServer() *Server {
	return &Server{
		port: ":8080",
	}
}

func (s *Server) initialize() error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if len(os.Args) > 1 {
		s.scannedPath = os.Args[1]
	} else {
		return fmt.Errorf("no path provided")
	}
	if _, err := os.Stat(s.scannedPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", s.scannedPath)
	}
	absPath, err := filepath.Abs(s.scannedPath)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %v", err)
	}
	s.scannedPath = absPath
	log.Printf("Starting directory scanner for: %s", s.scannedPath)
	log.Printf("Server will be available at: http://localhost%s", s.port)
	return nil
}

func (s *Server) setupRoutes(r *gin.Engine) {
	r.GET("/*path", s.handleBrowse)
}

func (s *Server) handleBrowse(c *gin.Context) {
	requestedPath := c.Param("path")

	if strings.HasPrefix(requestedPath, "/static/") {
		staticPath := "." + requestedPath
		c.File(staticPath)
		return
	}

	if requestedPath == "/favicon.ico" {
		c.Status(http.StatusNoContent)
		return
	}

	cleanedPath := strings.TrimPrefix(requestedPath, "/")
	fullPath := filepath.Join(s.scannedPath, cleanedPath)

	if !s.isPathAllowed(fullPath) {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		c.String(http.StatusNotFound, "Path not found: %v", err)
		return
	}

	if !info.IsDir() {
		s.serveFile(c, fullPath)
		return
	}

	data, err := s.scanDirectory(fullPath)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error scanning directory: %v", err)
		return
	}

	relPath, _ := filepath.Rel(s.scannedPath, fullPath)
	breadcrumb := s.createBreadcrumb(relPath)

	templateData := gin.H{
		"Title":      "Публичные документы",
		"Data":       data,
		"Breadcrumb": breadcrumb,
	}

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error parsing template: %v", err)
		return
	}

	c.Header("Content-Type", "text/html")
	err = tmpl.Execute(c.Writer, templateData)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (s *Server) isPathAllowed(path string) bool {
	return strings.HasPrefix(path, s.scannedPath)
}

func (s *Server) serveFile(c *gin.Context, fullPath string) error {
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("file not found")
	}
	if info.IsDir() {
		return fmt.Errorf("cannot view directory")
	}
	ext := strings.ToLower(filepath.Ext(fullPath))
	if ext == ".pdf" {
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "inline; filename="+filepath.Base(fullPath))
	}
	c.File(fullPath)
	return nil
}

func (s *Server) scanDirectory(dirPath string) (*DirectoryData, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path: %v", err)
	}
	if err := s.validateDirectory(absPath); err != nil {
		return nil, err
	}

	var files []FileInfo
	var directories []FileInfo
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	for _, d := range entries {
		path := filepath.Join(absPath, d.Name())
		fileInfo, err := s.createFileInfo(path, d)
		if err != nil {
			return nil, err
		}
		if d.IsDir() {
			directories = append(directories, fileInfo)
		} else {
			files = append(files, fileInfo)
		}
	}

	if absPath == s.scannedPath {
		// files = []FileInfo{} - This logic is no longer needed with server-side rendering
	}

	s.sortFileLists(files, directories)
	relPath := s.getRelativePath(absPath)
	return &DirectoryData{
		Path:        relPath,
		Files:       files,
		Directories: directories,
		TotalFiles:  len(files),
		TotalDirs:   len(directories),
		ScanTime:    time.Now(),
	}, nil
}

func (s *Server) validateDirectory(absPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("error accessing path: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}
	return nil
}

func (s *Server) createFileInfo(path string, d fs.DirEntry) (FileInfo, error) {
	relPath, err := filepath.Rel(s.scannedPath, path)
	if err != nil {
		return FileInfo{}, err
	}
	info, err := d.Info()
	if err != nil {
		return FileInfo{}, err
	}
	fileInfo := FileInfo{
		Name:         d.Name(),
		Path:         path,
		IsDir:        d.IsDir(),
		Size:         info.Size(),
		ModTime:      info.ModTime(),
		RelativePath: relPath,
	}
	if !d.IsDir() {
		fileInfo.Extension = strings.ToLower(filepath.Ext(d.Name()))
		fileInfo.CanView = s.canViewFile(fileInfo.Extension)
	} else {
		fileInfo.CanView = false
	}
	return fileInfo, nil
}

func (s *Server) sortFileLists(files []FileInfo, directories []FileInfo) {
	sort.Slice(directories, func(i, j int) bool {
		return directories[i].Name < directories[j].Name
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
}

func (s *Server) getRelativePath(absPath string) string {
	relPath, _ := filepath.Rel(s.scannedPath, absPath)
	if relPath == "." {
		relPath = "Root"
	}
	return relPath
}

func (s *Server) canViewFile(ext string) bool {
	viewableExtensions := map[string]bool{
		".pdf":  true,
		".txt":  true,
		".md":   true,
		".html": true,
		".htm":  true,
	}
	return viewableExtensions[ext]
}

func (s *Server) run() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	s.setupRoutes(r)
	return r.Run(s.port)
}

func main() {
	server := NewServer()
	if err := server.initialize(); err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	if err := server.run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

type BreadcrumbPart struct {
	Name string
	Path string
}

func (s *Server) createBreadcrumb(path string) []BreadcrumbPart {
	var parts []BreadcrumbPart
	parts = append(parts, BreadcrumbPart{Name: "Главная", Path: "/"})
	if path == "." || path == "" {
		return parts
	}

	currentPath := ""
	for _, part := range strings.Split(path, "/") {
		currentPath = filepath.Join(currentPath, part)
		parts = append(parts, BreadcrumbPart{Name: part, Path: "/" + currentPath})
	}
	return parts
} 