package handles

import (
	"fmt"
	"net/http"
	"slices"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
)

type PageData struct {
	Batches []Batch
}

type Batch struct {
	Label     string
	Timelines []Timeline
}

type Timeline struct {
	Date    string
	Title   string // Optional, left blank below
	Slides  []Slide
	Winners []Winner
	AspectRatio  string
}

type Slide struct {
	URL     string
	Caption string
}

type Winner struct {
	Tier  string // "gold", "silver", "bronze", "female"
	Label string // "1st", "2nd", "3rd", "Best female performer"
	Name  string
}

// --- Helper function to auto-generate the image URLs ---
// x = batch year (3, 4, or 5)
// y = event number (1 or 2)
// maxZ = number of photos (6 or 5)
func generateSlides(projectRef string, folderName string, x, y, maxZ int) []Slide {
	var slides []Slide
	baseURL := fmt.Sprintf("https://%s.supabase.co/storage/v1/object/public/%s", projectRef, folderName)

	for z := 1; z <= maxZ; z++ {
		slides = append(slides, Slide{
			URL:     fmt.Sprintf("%s/%d_%d_%d.jpg", baseURL, x, y, z),
			Caption: fmt.Sprintf("Batch %d - Event %d - Photo %d", 2020+x, y, z),
		})
	}
	return slides
}

// --- Main Handler ---

func PastEvents(c *gin.Context, cfg *configs.Config) {
	// Extract your Supabase config details
	supaBaseRef := cfg.SupaBase // e.g., "xxxx"
	folderName := cfg.FolderName

	// Below is the editable data structure. 
	// Edit the Date and Name fields manually.
	batches := []Batch{
		{
			Label: "2023", // x = 3
			Timelines: []Timeline{
				{
					Date:   "30th August, 2024", // y = 1
					AspectRatio: "1/1",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 3, 1, 5), // 6 photos
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Abhishek Jawanpuria"},
						{Tier: "silver", Label: "2nd", Name: "Ayush Agarwal"},
						{Tier: "bronze", Label: "3rd", Name: "Sidhant Singh"},
						{Tier: "female", Label: "Best female performer", Name: "Puja Rani"},
					},
				},
				{
					Date:   "2nd February, 2025", // y = 2
					AspectRatio: "1/1",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 3, 2, 5), // 5 photos
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Daksh Panwar"},
						{Tier: "silver", Label: "2nd", Name: "Shubham Pandey"},
						{Tier: "bronze", Label: "3rd", Name: "Sidhant Singh"},
					},
				},
			},
		},
		{
			Label: "2024", // x = 4
			Timelines: []Timeline{
				{
					Date:   "12th April, 2025", // y = 1
					AspectRatio: "16/9",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 4, 1, 6), // 6 photos
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Adarsh Raj"},
						{Tier: "silver", Label: "2nd", Name: "Amit Kumar"},
						{Tier: "bronze", Label: "3rd", Name: "Ashray Saxena"},
						{Tier: "female", Label: "Best female performer", Name: "Shristi"},
					},
				},
				{
					Date:   "6th September, 2025", // y = 2
					AspectRatio: "16/9",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 4, 2, 5), // 5 photos
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Priyanshu"},
						{Tier: "silver", Label: "2nd", Name: "Adarsh Raj"},
						{Tier: "bronze", Label: "3rd", Name: "Harsh Gautam"},
					},
				},
			},
		},
		{
			Label: "2025", // x = 5
			Timelines: []Timeline{
				{
					Date:   "18th April, 2026", // y = 1
					AspectRatio: "16/9",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 5, 1, 6), // 6 photos
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Ujjawal Kumar"},
						{Tier: "silver", Label: "2nd", Name: "Prashant Sharma"},
						{Tier: "bronze", Label: "3rd", Name: "Chinmay"},
						{Tier: "female", Label: "Best female performer", Name: "Isha Roy"},
					},
				},
			},
		},
	}

	slices.Reverse(batches)
	
	// pageData := PageData{
	// 	Batches: batches,
	// }

	//c.HTML(http.StatusOK, "events.tmpl", pageData)
	// c.HTML(http.StatusOK, "events.tmpl", gin.H{
	// 	"Batches": pageData,
	// })
	c.HTML(http.StatusOK, "events.tmpl", gin.H{
		"Batches": batches,
	})
}