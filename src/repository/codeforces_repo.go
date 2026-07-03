package repository

import (
	"leaderboard/src/database"
	"log"
)

// returns all codeforces handles for past users
func GetPastUserHandles() ([]string, error) {
	rows, err := database.DB.Query("SELECT codeforces_handle FROM past_users")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var handles []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		handles = append(handles, h)
	}
	return handles, nil
}

// updates a past user's rating stats
func UpdatePastUserRating(rating, maxRating int, title, handle string) error {
	_, err := database.DB.Exec(`
		UPDATE past_users
		SET current_rating = ?,
			max_rating = ?,
			title = ?,
			last_updated = CURRENT_TIMESTAMP
		WHERE codeforces_handle = ?
	`, rating, maxRating, title, handle)
	return err
}

// returns a list of users for dropdowns
func GetUsersList() []map[string]interface{} {
	rows, err := database.DB.Query("SELECT codeforces_handle, display_name FROM users")
	if err != nil {
		return nil
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var users []map[string]interface{}
	for rows.Next() {
		var handle, displayName string
		if err := rows.Scan(&handle, &displayName); err != nil {
			log.Printf("failed to scan close row : %v", err)
		}
		users = append(users, map[string]interface{}{"Username": handle, "DisplayName": displayName})
	}
	return users
}
