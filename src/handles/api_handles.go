package handles

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"leaderboard/src/configs"
	"leaderboard/src/database"
	"leaderboard/src/workers"
	"log"
	"net/http"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type DefaultHTTPClient struct{}

func (*DefaultHTTPClient) Get(url string) (*http.Response, error) {
	return http.Get(url)
}

type APIHandler struct {
	client HTTPClient
	cfg    *configs.Config
}

func NewAPIHandler(client HTTPClient, cfg *configs.Config) *APIHandler {
	return &APIHandler{
		client: client,
		cfg:    cfg,
	}
}

func waitForCFLimit(start time.Time) {
	elapsed := time.Since(start)

	if elapsed < 2*time.Second {
		time.Sleep(2*time.Second - elapsed)
	}
}

func (h *APIHandler) CheckCFAPI(c *gin.Context) {
	start := time.Now()

	url := "https://codeforces.com/api/system.status"

	resp, err := h.client.Get(url)
	if err != nil {
		waitForCFLimit(start)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Codeforces API unreachable",
		})
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		waitForCFLimit(start)
		c.JSON(http.StatusBadGateway, gin.H{
			"status":  "error",
			"message": "CF returned non-200",
			"http":    resp.StatusCode,
			"body":    string(body),
		})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		waitForCFLimit(start)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Invalid JSON from CF",
		})
		return
	}

	if result["status"] != "OK" {
		waitForCFLimit(start)
		c.JSON(http.StatusBadGateway, gin.H{
			"status":  "error",
			"message": "CF API status not OK",
		})
		return
	}

	waitForCFLimit(start)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Codeforces API is alive",
	})
}

func SendPing(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func (h *APIHandler) Purg(c *gin.Context) {
	clientToken := c.GetHeader("X-Cron-Token")

	if clientToken == "" || clientToken != h.cfg.CronSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	if err := workers.PurgeAsynqMetadata(database.RedisClient); err != nil {
		log.Printf("Redis optimization sweep failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to purge Redis metadata",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Redis metadata purged and optimized successfully",
	})
}
