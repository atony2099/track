package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"track/db"
)

var (
	date    string
	numDays int
)

// 22
// 33
var (
	conn *db.Database
)

type InputParams struct {
	Tag       string
	StartTime string
	EndTime   string
}

// 22
func main() {

	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "-h") {
		printHelp()
		os.Exit(0)
	}

	var err error
	conn, err = db.NewDatabase()
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}

	cmd := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	switch os.Args[1] {

	case "timeline":

		cmd.IntVar(&numDays, "n", 0, "Number of days")
		cmd.StringVar(&date, "d", "", "Date")
		// fmt.Println("os.Args", os.Args[2:], len(os.Args), cap(os.Args))
		cmd.Parse(os.Args[2:])
		listTimelines(getDates())

	case "percent":

		cmd.IntVar(&numDays, "n", 0, "Number of days")
		cmd.StringVar(&date, "d", "", "Date")

		cmd.Parse(os.Args[2:])
		listPercent(getDates())

	case "fill":
		cmd.StringVar(&date, "d", "", "Date")
		cmd.IntVar(&numDays, "n", 0, "offset day")
		cmd.Parse(os.Args[2:])

		date := getDate()
		listTimelines([]string{date})
		fillMissingActivities(date)

	case "input":

		var rang, tag string

		cmd.StringVar(&rang, "range", "", "the time range")
		cmd.StringVar(&tag, "tag", "", "the tag of activity")
		cmd.Parse(os.Args[2:])
		inputActivity(rang, tag)

	default:
		fmt.Println("Invalid command")
		cmd.Usage()
		os.Exit(1)
	}

}

func inputActivity(rang string, tag string) {

	var activity db.Activity

	if tag == "" {
		tag = "distraction"
	}

	activity.Date = time.Now().Format("2006-01-02")
	activity.Tags = tag
	startEndTime := strings.Split(rang, "-")

	activity.StartTime = startEndTime[0]
	activity.EndTime = startEndTime[1]

	fmt.Println("startEndTime:", startEndTime, "len:", len(startEndTime), tag)
	// 记录到数据库
	err := conn.InsertActivity(activity)

	if err != nil {
		log.Fatalf("error inserting activity: %v", err)

	}
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  command [flags]")
	fmt.Println("Commands:")
	fmt.Println("  timeline  View timeline over date range")
	fmt.Println("  percent   View percentage tracked over date range")
	fmt.Println("  fill      Fill in any missing dates")
}

func getDate() string {
	var dateStr string

	fmt.Println("date:", date, "numDays:", numDays, "")

	if date != "" {
		parsed, err := time.Parse("20060102", date)
		if err != nil {
			log.Fatal(err)
		}

		dateStr = parsed.Format(time.DateOnly)

	} else if numDays > 0 {
		dateStr = time.Now().AddDate(0, 0, -numDays).Format(time.DateOnly)

	} else {
		dateStr = time.Now().Format(time.DateOnly)
		// dates = append(dates, time.Now().Format(time.DateOnly))
	}

	// fmt.Println("dates:", dates, "number of days:", numDays, "date:", date, "")
	return dateStr
}

func getDates() []string {
	var dates []string

	if date != "" {
		parsed, err := time.Parse("20060102", date)
		if err != nil {
			log.Fatal(err)
		}
		dates = append(dates, parsed.Format(time.DateOnly))
	} else if numDays > 0 {
		for i := 0; i < numDays; i++ {
			dates = append(dates, time.Now().AddDate(0, 0, -i).Format(time.DateOnly))
		}
	} else {
		dates = append(dates, time.Now().Format(time.DateOnly))
	}

	// fmt.Println("dates:", dates, "number of days:", numDays, "date:", date, "")
	return dates
}

// Rest of functions

func fillMissingActivities(date string) {
	fmt.Printf("\n%s  ", date)

fill:
	var startTime = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)

	var endTime time.Time
	if date == time.Now().Format("2006-01-02") {
		now := time.Now()
		endTime = time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
	} else {
		endTime = startTime.Add(24*time.Hour - time.Second)
	}

	// fmt.Println("startTime:", startTime, "endTime:", endTime, "")

	activities, _ := conn.GetActivitiesByDate(date)

	lastEndTime := startTime
	for _, activity := range activities {

		activityStartTime, _ := time.Parse("15:04:05", activity.StartTime)

		if activityStartTime.After(lastEndTime) {

			// fmt.Println("lastEndTime:", lastEndTime, "activityStartTime:", activityStartTime, activityStartTime.Sub(lastEndTime))

			change := promptForActivity(date, lastEndTime, activityStartTime)
			if change {
				goto fill
			}
		}
		lastEndTime, _ = time.Parse("15:04:05", activity.EndTime)

	}

	if endTime.After(lastEndTime) {
		// fmt.Printf("start time %v, end time %v \n", lastEndTime, endTime)
		// fmt.Println("lastEndTime--:", endTime, "activityStartTime:", lastEndTime, endTime.Sub(lastEndTime))
		change := promptForActivity(date, lastEndTime, endTime)
		if change {
			goto fill
		}

	}

	listTimelineDetail(date)
}

//0001-01-01 00:00:00 +0000 UTC,
//0000-01-01 18:30:02.467646 +0000 UTC

// Prompt user for activity between start and end times
func promptForActivity(date string, startTime, endTime time.Time) bool {

	fmt.Printf("[%s - %s], total %v \n", startTime.Format("15:04:05"), endTime.Format("15:04:05"), endTime.Sub(startTime))

	var activity db.Activity
	activity.Date = date
	reader := bufio.NewReader(os.Stdin)
	activity.Tags = getTag(reader)
	var changeEndTime bool

	var start, end string

	fmt.Print("enter [hhmmss-hhmmss]  or use leave blank to use default: ")
	startEnd, _ := reader.ReadString('\n')
	startEnd = strings.TrimSpace(startEnd)
	startEnds := strings.Split(startEnd, "-")

	if len(startEnds) == 2 {
		start = startEnds[0]
		end = startEnds[1]
	} else {
		start = startEnds[0]
	}

	if start == "" {
		start = startTime.Format("15:04:05")
	} else {
		start = start[:2] + ":" + start[2:]
		changeEndTime = true
	}
	activity.StartTime = start

	if end == "" {
		end = endTime.Format("15:04:05")
	} else {
		end = end[:2] + ":" + end[2:]
		changeEndTime = true
	}
	activity.EndTime = end

	conn.InsertActivity(activity)

	return changeEndTime
}

// Get activities for a date

// Insert new activity into database

func listTimelines(dates []string) {
	for _, date := range dates {
		listTimelineDetail(date)
	}
}

func listTimelineDetail(date string) {

	list, err := conn.GetActivitiesByDate(date)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Activities for", date)
	fmt.Printf("%-10s %-20s  %-10s %-10s %-10s\n", "Date", "Tags", "Start Time", "End Time", "Duration")
	for _, activity := range list {

		// Calculate duration
		start, err := time.Parse("15:04:05", activity.StartTime)
		if err != nil {
			log.Fatal(err)
		}
		end, err := time.Parse("15:04:05", activity.EndTime)
		if err != nil {
			log.Fatal(err)
		}
		duration := newFunction(end, start)

		// Print the activity with aligned fields
		fmt.Printf("%-10s %-20s  %-10s %-10s %-10s\n", activity.Date, activity.Tags, activity.StartTime, activity.EndTime, duration)

	}

}

func newFunction(end time.Time, start time.Time) time.Duration {
	duration := end.Sub(start)
	return duration
}

func track() {
	var activity db.Activity
	var day string = time.Now().Format("2006-01-02")

	activity.Date = day

	reader := bufio.NewReader(os.Stdin)
	var lastEndTime = ""

	fmt.Print("Enter tags (use numbers for existing tags): ")
	activity.Tags = getTag(reader)

	activ, err := conn.GetLatestActivityByDate(day)

	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("error fetching last activity %v", err)
	}

	if err == sql.ErrNoRows {
		lastEndTime = "00:00:00"
	} else {
		lastEndTime = activ.EndTime
	}

	fmt.Printf("Start time (HHMMSS) OR %v: ", lastEndTime)
	start, _ := reader.ReadString('\n')
	start = strings.TrimSpace(start)

	if start == "" {
		start = lastEndTime
	} else {
		start = start[:2] + ":" + start[2:]
	}

	activity.StartTime = strings.TrimSpace(start)

	now := time.Now().Format("15:04:05")
	fmt.Printf("End time (HHMMSS) OR %v ", now)
	end, _ := reader.ReadString('\n')
	end = strings.TrimSpace(end)

	if end != "" {
		activity.EndTime = end[:2] + ":" + end[2:]
	} else {
		activity.EndTime = now
	}

	if activity.Tags == "" {
		activity.Tags = "play"
	}

	err = conn.InsertActivity(activity)

	if err != nil {
		log.Fatalf("error inserting activity %v", err)
	}

	listTimelineDetail(day)
}

func getTag(reader *bufio.Reader) string {

	choices, err := conn.GetAllTags()
	if err != nil {
		log.Fatalf("error fetching distinct values %v", err)
	}
	sort.Strings(choices)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return "play"
	}

	if num, err := strconv.Atoi(input); err == nil && num > 0 && num <= len(choices) {
		return choices[num-1]
	}

	for _, v := range choices {
		if strings.HasPrefix(v, input) {
			return v
		}
	}
	return input
}

func listPercent(dates []string) {

	results, err := conn.ListActivitiesByTag(dates)

	if err != nil {
		log.Fatalf("error fetching results %v", err)
	}

	var totalHours float64 = 24 * float64(len(dates))
	maxTagLength := 0
	for _, v := range results {
		tagLength := len(v.Tags)
		if tagLength > maxTagLength {
			maxTagLength = tagLength
		}
	}

	for i := 0; i < len(results); i++ {

		tag := results[i].Tags
		duration := results[i].Duration

		hours := duration.Hours()
		// fmt.Println("hours:", hours, "")
		percentage := 100 * hours / totalHours

		tagPadding := strings.Repeat(" ", maxTagLength-len(tag))

		fmt.Printf("%s%s %5.2f hours (%4.1f%%)",
			tag, tagPadding, hours, percentage)

		printProgressBar(percentage / 100)

		fmt.Println()
	}
}

func printProgressBar(progress float64) {

	totalBars := 20

	// 实心部分用 █
	filled := int(progress * float64(totalBars))
	for i := 0; i < filled; i++ {
		fmt.Print("█")
	}

	// 未填充部分用 ░
	remaining := totalBars - filled
	for i := 0; i < remaining; i++ {
		fmt.Print("░")
	}

	fmt.Print(" ")

	percent := int(progress * 100)
	fmt.Printf("%d%%\n", percent)

}
