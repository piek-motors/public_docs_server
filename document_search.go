package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DocumentIndex represents the in-memory index of documents
type DocumentIndex struct {
	mu       sync.RWMutex
	documents map[string][]DocumentInfo
	lastScan time.Time
}
// DocumentInfo represents information about a found document
type DocumentInfo struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	FullPath string `json:"full_path"`
}
// SearchResult represents the result of a document search
type SearchResult struct {
	Query     string         `json:"query"`
	Results   []DocumentInfo `json:"results"`
	Count     int            `json:"count"`
	SearchTime time.Time     `json:"search_time"`
}
// NewDocumentIndex creates a new document index
func NewDocumentIndex() *DocumentIndex {
	return &DocumentIndex{
		documents: make(map[string][]DocumentInfo),
		lastScan:  time.Time{},
	}
}
// StartIndexing starts the background indexing process
func (di *DocumentIndex) StartIndexing(rootPath string) {
	go func() {
		// Initial scan
		di.scanDocuments(rootPath)
		// Periodic refresh every 10 minutes
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			di.scanDocuments(rootPath)
		}
	}()
}
// scanDocuments scans the directory recursively and indexes documents
func (di *DocumentIndex) scanDocuments(rootPath string) {
	di.mu.Lock()
	defer di.mu.Unlock()
	// Clear existing index
	di.documents = make(map[string][]DocumentInfo)
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue walking
		}
		if info.IsDir() {
			return nil // Skip directories
		}
		docName := info.Name()
		if docName != "" {
			relPath, _ := filepath.Rel(rootPath, path)
			docInfo := DocumentInfo{
				ID:       docName,
				Path:     relPath,
				Name:     info.Name(),
				Size:     info.Size(),
				ModTime:  info.ModTime(),
				FullPath: path,
			}
			di.documents[docName] = append(di.documents[docName], docInfo)
		}
		return nil
	})
	if err != nil {
		log.Printf("Error during document scan: %v", err)
		return
	}
	di.lastScan = time.Now()
}

// SearchDocuments searches for documents by ID
func (di *DocumentIndex) SearchDocuments(query string) *SearchResult {
	di.mu.RLock()
	defer di.mu.RUnlock()
	query = strings.TrimSpace(query)
	if query == "" {
		return &SearchResult{
			Query:      query,
			Results:    []DocumentInfo{},
			Count:      0,
			SearchTime: time.Now(),
		}
	}
	var results []DocumentInfo
	for docID, docs := range di.documents {
		if strings.HasPrefix(docID, query) {
			results = append(results, docs...)
		}
	}
	return &SearchResult{
		Query:      query,
		Results:    results,
		Count:      len(results),
		SearchTime: time.Now(),
	}
}
// GetIndexStats returns statistics about the document index
func (di *DocumentIndex) GetIndexStats() map[string]interface{} {
	di.mu.RLock()
	defer di.mu.RUnlock()
	totalDocs := 0
	for _, docs := range di.documents {
		totalDocs += len(docs)
	}
	return map[string]interface{}{
		"unique_ids":    len(di.documents),
		"total_files":   totalDocs,
		"last_scan":     di.lastScan,
		"index_age":     time.Since(di.lastScan).String(),
	}
}
// ForceRefresh forces an immediate refresh of the document index
func (di *DocumentIndex) ForceRefresh(rootPath string) {
	log.Printf("Forcing document index refresh")
	di.scanDocuments(rootPath)
} 