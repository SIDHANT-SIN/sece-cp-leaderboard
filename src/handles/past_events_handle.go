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
	Title   string 
	Slides  []Slide
	Winners []Winner
	AspectRatio  string
}

type Slide struct {
	URL     string
	Caption string
}

type Winner struct {
	Tier  string 
	Label string 
	Name  string
}

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


func PastEvents(c *gin.Context, cfg *configs.Config) {
	supaBaseRef := cfg.SupaBase 
	folderName := cfg.FolderName

	batches := []Batch{
		{
			Label: "2023", 
			Timelines: []Timeline{
				{
					Date:   "30th August, 2024", 
					AspectRatio: "1/1",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 3, 1, 5), 
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Abhishek Jawanpuria"},
						{Tier: "silver", Label: "2nd", Name: "Ayush Agarwal"},
						{Tier: "bronze", Label: "3rd", Name: "Sidhant Singh"},
						{Tier: "female", Label: "Best female performer", Name: "Puja Rani"},
					},
				},
				{
					Date:   "2nd February, 2025", 
					AspectRatio: "1/1",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 3, 2, 5), 
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Daksh Panwar"},
						{Tier: "silver", Label: "2nd", Name: "Shubham Pandey"},
						{Tier: "bronze", Label: "3rd", Name: "Sidhant Singh"},
					},
				},
			},
		},
		{
			Label: "2024", 
			Timelines: []Timeline{
				{
					Date:   "12th April, 2025", 
					AspectRatio: "16/9",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 4, 1, 6), 
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Adarsh Raj"},
						{Tier: "silver", Label: "2nd", Name: "Amit Kumar"},
						{Tier: "bronze", Label: "3rd", Name: "Ashray Saxena"},
						{Tier: "female", Label: "Best female performer", Name: "Shristi"},
					},
				},
				{
					Date:   "6th September, 2025", 
					AspectRatio: "16/9",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 4, 2, 5),
					Winners: []Winner{
						{Tier: "gold", Label: "1st", Name: "Priyanshu"},
						{Tier: "silver", Label: "2nd", Name: "Adarsh Raj"},
						{Tier: "bronze", Label: "3rd", Name: "Harsh Gautam"},
					},
				},
			},
		},
		{
			Label: "2025", 
			Timelines: []Timeline{
				{
					Date:   "18th April, 2026", 
					AspectRatio: "16/9",
					Title:  "",
					Slides: generateSlides(supaBaseRef, folderName, 5, 1, 6), 
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

	c.HTML(http.StatusOK, "events.tmpl", gin.H{
		"Batches": batches,
	})
}