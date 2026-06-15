package handles


import (
	"encoding/json"
	"io"
	
	"net/http"

	"github.com/gin-gonic/gin"

	"time"

	// "bytes"
	// "os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)


func waitForCFLimit(start time.Time) {
	elapsed := time.Since(start)

	if elapsed < 2*time.Second {
		time.Sleep(2*time.Second - elapsed)
	}
}

func CheckCFAPI(c *gin.Context) {
	start := time.Now()

	url := "https://codeforces.com/api/system.status"

	resp, err := http.Get(url)
	if err != nil {
		waitForCFLimit(start)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Codeforces API unreachable",
		})
		return
	}
	defer resp.Body.Close()

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
