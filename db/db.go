// track/data/db.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Database struct {
	Conn *sql.DB
}

func NewDatabase() (*Database, error) {
	dbUser := os.Getenv("MYSQL_USER")
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	dbName := "time"
	dbHost := "127.0.0.1"
	dbPort := "3306"

	conn, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName))
	if err != nil {
		return nil, err
	}
	return &Database{Conn: conn}, nil
}

type Activity struct {
	Date      string
	Tags      string
	StartTime string
	EndTime   string
}

type Result struct {
	Tags     string
	Duration time.Duration
}

func (db *Database) GetLatestActivityByDate(data string) (Activity, error) {

	query := `
		SELECT date,tags,start_time,end_time FROM daily_tracker 
		WHERE date = ?
		ORDER BY start_time DESC
		LIMIT 1
	`

	var activity Activity
	err := db.Conn.QueryRow(query, data).Scan(&activity.Date, &activity.Tags, &activity.StartTime, &activity.EndTime)
	if err != nil {
		return Activity{}, err
	}
	return activity, nil

}

func (db *Database) GetActivitiesByDate(date string) ([]Activity, error) {

	query := `
    SELECT date,tags,start_time,end_time FROM daily_tracker 
    WHERE date = ?
    ORDER BY start_time
  `

	rows, err := db.Conn.Query(query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []Activity{}

	for rows.Next() {
		var activity Activity
		err := rows.Scan(&activity.Date, &activity.Tags, &activity.StartTime, &activity.EndTime)
		if err != nil {
			log.Fatalf("Error scanning row: %v", err)
		}

		activities = append(activities, activity)
	}

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].StartTime < activities[j].StartTime
	})

	return activities, nil

}

func (db *Database) InsertActivity(activity Activity) error {

	stmt := `
    INSERT INTO daily_tracker (date, tags, start_time, end_time)
    VALUES (?, ?, ?, ?)
  `

	_, err := db.Conn.Exec(stmt, activity.Date, activity.Tags, activity.StartTime, activity.EndTime)
	return err

}

func (db *Database) ListActivitiesByTag(dates []string) ([]Result, error) {

	dateStr := "'" + strings.Join(dates, "','") + "'"

	query := fmt.Sprintf("SELECT tags, SEC_TO_TIME(SUM(TIME_TO_SEC(end_time) - TIME_TO_SEC(start_time))) AS total_time FROM daily_tracker WHERE `date` IN (%s)  GROUP BY tags", dateStr)
	rows, err := db.Conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var tag string
		var timeStr string
		if err := rows.Scan(&tag, &timeStr); err != nil {
			return nil, err
		}

		parts := strings.Split(timeStr, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid time format")
		}

		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}

		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}

		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
		}

		duration := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second

		results = append(results, Result{tag, duration})

	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Duration > results[j].Duration
	})

	return results, nil

}

func (db *Database) GetAllTags() ([]string, error) {
	rows, err := db.Conn.Query(fmt.Sprintf("SELECT DISTINCT %s FROM daily_tracker", "tags"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	i := 1
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		fmt.Printf("%d. %s ", i, value)
		values = append(values, value)
		i++
	}
	return values, nil
}
