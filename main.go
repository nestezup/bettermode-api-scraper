package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "gpters_scrap/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title BetterMode API Scraper
// @version 1.0
// @description A server that retrieves content from BetterMode API
// @BasePath /api/v1

// @license.name MIT
// @host localhost:8080
// @schemes http

// TokenManager 구조체는 BetterMode API 토큰을 관리합니다
type TokenManager struct {
	accessToken   string
	expiry        time.Time
	networkDomain string
	mutex         sync.RWMutex
}

// NewTokenManager는 TokenManager 인스턴스를 생성하고 초기화합니다
func NewTokenManager(networkDomain string) *TokenManager {
	tm := &TokenManager{
		networkDomain: networkDomain,
	}
	// 초기 토큰 가져오기
	err := tm.RefreshToken()
	if err != nil {
		log.Printf("Initial token fetch failed: %v. Will retry on first request.", err)
	}
	return tm
}

// GetToken은 현재 유효한 액세스 토큰을 반환합니다. 필요한 경우 갱신합니다.
func (tm *TokenManager) GetToken() (string, error) {
	tm.mutex.RLock()
	// 토큰이 없거나 곧 만료될 예정이면 (5분 이내)
	if tm.accessToken == "" || time.Now().Add(5*time.Minute).After(tm.expiry) {
		tm.mutex.RUnlock()
		err := tm.RefreshToken()
		if err != nil {
			return "", err
		}
		tm.mutex.RLock()
	}
	token := tm.accessToken
	tm.mutex.RUnlock()
	return token, nil
}

// RefreshToken은 BetterMode API에서 새 게스트 액세스 토큰을 가져옵니다
func (tm *TokenManager) RefreshToken() error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// API 요청을 위한 GraphQL 쿼리
	query := map[string]interface{}{
		"query": `
			query {
				tokens(networkDomain: "www.gpters.org") {
					accessToken
				}
			}
		`,
	}

	jsonBody, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("error marshalling token query: %w", err)
	}

	// API 요청 생성
	req, err := http.NewRequest("POST", "https://api.bettermode.com/", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 요청 전송
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending token request: %w", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading token response: %w", err)
	}

	// 응답 파싱
	var tokenResponse struct {
		Data struct {
			Tokens struct {
				AccessToken string `json:"accessToken"`
			} `json:"tokens"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return fmt.Errorf("error parsing token response: %w", err)
	}

	if tokenResponse.Data.Tokens.AccessToken == "" {
		return fmt.Errorf("no token returned from API")
	}

	// 토큰 저장
	tm.accessToken = tokenResponse.Data.Tokens.AccessToken

	// JWT 토큰에서 만료 시간 추출 (선택 사항, 구현에 따라 다를 수 있음)
	// 만료 시간을 확인할 수 없는 경우 24시간으로 설정
	tm.expiry = time.Now().Add(24 * time.Hour)

	log.Printf("Token refreshed successfully, valid until %v", tm.expiry)
	return nil
}

type PostResponse struct {
	Data struct {
		Post struct {
			MappingFields []struct {
				Key   string `json:"key"`
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"mappingFields"`
			Title string `json:"title"`
		} `json:"post"`
	} `json:"data"`
}

type ContentRequest struct {
	PostID string `json:"post_id"`
	Format string `json:"format,omitempty"` // "html" (default) or "text"
}

type ContentResponse struct {
	Content   string `json:"content"`
	Format    string `json:"format"`
	PostID    string `json:"post_id"`
	Title     string `json:"title,omitempty"`
	CharCount int    `json:"char_count,omitempty"`
}

// 전역 토큰 관리자
var tokenManager *TokenManager

// GetContent godoc
// @Summary Get content from BetterMode API
// @Description Retrieves content value from mappingFields where key is "content"
// @Tags content
// @Accept json
// @Produce json
// @Param request body ContentRequest true "Post ID and optional format (html or text)"
// @Success 200 {object} ContentResponse
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Internal server error"
// @Router /content [post]
func getContent(w http.ResponseWriter, r *http.Request) {
	var req ContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PostID == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	// Set default format to html if not specified
	if req.Format == "" {
		req.Format = "html"
	} else if req.Format != "html" && req.Format != "text" {
		http.Error(w, "Format must be 'html' or 'text'", http.StatusBadRequest)
		return
	}

	// Fetch content and title
	content, title, err := fetchContentFromBetterMode(req.PostID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching content: %v", err), http.StatusInternalServerError)
		return
	}

	// Clean up the content value
	processedContent := cleanupContent(content)

	// If format is text, try to strip HTML tags
	if req.Format == "text" {
		processedContent = stripHTMLTags(processedContent)
	}

	// Prepare the response
	response := ContentResponse{
		Content:   processedContent,
		Format:    req.Format,
		PostID:    req.PostID,
		Title:     title,
		CharCount: len(processedContent),
	}

	render.JSON(w, r, response)
}

func fetchContentFromBetterMode(postID string) (string, string, error) {
	url := "https://api.bettermode.com/"

	// 토큰 관리자에서 유효한 토큰 얻기
	token, err := tokenManager.GetToken()
	if err != nil {
		return "", "", fmt.Errorf("error getting access token: %w", err)
	}

	// Create the GraphQL query
	query := map[string]interface{}{
		"query": `query GetPost($id: ID!) {
			post(id: $id) {
				mappingFields {
					key
					type
					value
				}
				title
			}
		}`,
		"variables": map[string]interface{}{
			"id": postID,
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return "", "", fmt.Errorf("error marshalling query: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(queryJSON))
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	// Set headers with dynamic token
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "GPTers-Scraper/1.0")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check for unauthorized response (token might be expired)
	if resp.StatusCode == http.StatusUnauthorized {
		// Force token refresh and retry once
		log.Println("Token seems expired, refreshing and retrying...")
		err := tokenManager.RefreshToken()
		if err != nil {
			return "", "", fmt.Errorf("failed to refresh token: %w", err)
		}

		// Retry with new token
		return fetchContentFromBetterMode(postID)
	}

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading response: %w", err)
	}

	// Parse the response
	var postResp PostResponse
	if err := json.Unmarshal(body, &postResp); err != nil {
		return "", "", fmt.Errorf("error parsing response: %w", err)
	}

	// Get the title
	title := postResp.Data.Post.Title

	// Find the content field
	var content string
	for _, field := range postResp.Data.Post.MappingFields {
		if field.Key == "content" {
			content = field.Value
			break
		}
	}

	if content == "" {
		return "", title, fmt.Errorf("content field not found")
	}

	return content, title, nil
}

// cleanupContent cleans up HTML and escaped characters in the content
func cleanupContent(content string) string {
	// Remove the surrounding quotes if they exist
	if len(content) > 2 && content[0] == '"' && content[len(content)-1] == '"' {
		content = content[1 : len(content)-1]
	}

	// Replace escaped quotes with regular quotes
	content = strings.ReplaceAll(content, "\\\"", "\"")

	// Decode escaped Unicode characters
	var result string
	var err error

	// Attempt JSON unescaping first
	if result, err = unescapeUnicodeJSON(content); err == nil {
		content = result
	}

	// Replace common HTML entities with their characters
	htmlReplacements := map[string]string{
		"&nbsp;": " ",
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
		"&apos;": "'",
	}

	for escaped, unescaped := range htmlReplacements {
		content = strings.ReplaceAll(content, escaped, unescaped)
	}

	return content
}

// unescapeUnicodeJSON unescapes Unicode sequences in JSON strings
func unescapeUnicodeJSON(s string) (string, error) {
	// Create a temporary JSON string with the content as the value
	jsonStr := fmt.Sprintf(`{"content": %s}`, strconv.Quote(s))

	// Unmarshal to decode all escaped characters
	var result struct {
		Content string `json:"content"`
	}

	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return s, err
	}

	return result.Content, nil
}

// stripHTMLTags removes HTML tags from the content to provide plain text
func stripHTMLTags(html string) string {
	// Basic HTML tag removal
	var result strings.Builder
	var inTag bool

	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			// Add a space after closing tags for readability
			result.WriteRune(' ')
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// Remove extra spaces and normalize line breaks
	text := result.String()
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "\n\n", "\n")

	// Replace multiple spaces with a single space
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return strings.TrimSpace(text)
}

func main() {
	// 토큰 관리자 초기화
	tokenManager = NewTokenManager("www.gpters.org")

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// API Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/content", getContent)

		// 토큰 관리 엔드포인트 (관리자용) 추가
		r.Get("/token/refresh", handleTokenRefresh)
		r.Get("/token/status", handleTokenStatus)
	})

	// Swagger docs
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// handleTokenRefresh는 토큰을 수동으로 갱신하는 엔드포인트입니다 (관리자용)
func handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	// 실제 서비스에서는 관리자 인증 추가 필요
	err := tokenManager.RefreshToken()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to refresh token: %v", err), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, map[string]string{
		"status":  "success",
		"message": "Token refreshed successfully",
	})
}

// handleTokenStatus는 현재 토큰 상태를 확인하는 엔드포인트입니다 (관리자용)
func handleTokenStatus(w http.ResponseWriter, r *http.Request) {
	// 실제 서비스에서는 관리자 인증 추가 필요
	tokenManager.mutex.RLock()
	defer tokenManager.mutex.RUnlock()

	// 토큰의 처음 몇 글자만 공개
	tokenPreview := ""
	if len(tokenManager.accessToken) > 10 {
		tokenPreview = tokenManager.accessToken[:10] + "..."
	}

	render.JSON(w, r, map[string]interface{}{
		"status":        "success",
		"token_preview": tokenPreview,
		"expiry":        tokenManager.expiry,
		"is_valid":      time.Now().Before(tokenManager.expiry),
		"expires_in":    time.Until(tokenManager.expiry).String(),
	})
}
