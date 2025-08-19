package db

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Event struct {
	ID       int       `db:"id"`
	Name     string    `db:"name"`
	Date     time.Time `db:"date"`
	Location string    `db:"location"`
	Duration int       `db:"duration_hours"`
}

type UserSchedule struct {
	ID          int       `db:"id"`
	UserID      int       `db:"user_id"`
	UserName    string    `db:"user_name"`
	EventID     int       `db:"event_id"`
	WorkingFrom time.Time `db:"working_from"`
	WorkingTo   time.Time `db:"working_to"`
}

func InitDB() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", "./events.db")
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		date DATETIME NOT NULL,
		location TEXT NOT NULL,
		duration_hours INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS user_schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		user_name TEXT NOT NULL,
		event_id INTEGER NOT NULL,
		working_from DATETIME NOT NULL,
		working_to DATETIME NOT NULL,
		FOREIGN KEY (event_id) REFERENCES events(id)
	);`

	db.MustExec(schema)

	seedData(db)

	return db, nil
}

func seedData(db *sqlx.DB) {
	db.MustExec("DELETE FROM user_schedules")
	db.MustExec("DELETE FROM events")

	events := []Event{
		{Name: "Team Meeting", Date: time.Now().Add(24 * time.Hour), Location: "Conference Room A", Duration: 2},
		{Name: "Project Review", Date: time.Now().Add(24 * time.Hour), Location: "Conference Room B", Duration: 1},
		{Name: "Client Presentation", Date: time.Now().Add(48 * time.Hour), Location: "Main Hall", Duration: 3},
		{Name: "Training Session", Date: time.Now().Add(72 * time.Hour), Location: "Training Room", Duration: 4},
		// Add some conflicting events for testing venue overlap detection
		{Name: "Marketing Standup", Date: time.Now().Add(24*time.Hour + 30*time.Minute), Location: "Conference Room A", Duration: 1},
		{Name: "Board Meeting", Date: time.Now().Add(48*time.Hour + 1*time.Hour), Location: "Main Hall", Duration: 2},
	}

	for _, e := range events {
		db.MustExec(
			"INSERT INTO events (name, date, location, duration_hours) VALUES (?, ?, ?, ?)",
			e.Name, e.Date, e.Location, e.Duration,
		)
	}

	// Create time values for working hours
	today := time.Now().Truncate(24 * time.Hour) // Start of today
	
	schedules := []UserSchedule{
		{UserID: 1, UserName: "Alice", EventID: 1, WorkingFrom: today.Add(9 * time.Hour), WorkingTo: today.Add(17 * time.Hour)},
		{UserID: 1, UserName: "Alice", EventID: 2, WorkingFrom: today.Add(9 * time.Hour), WorkingTo: today.Add(17 * time.Hour)},
		{UserID: 2, UserName: "Bob", EventID: 1, WorkingFrom: today.Add(10 * time.Hour), WorkingTo: today.Add(18 * time.Hour)},
		{UserID: 2, UserName: "Bob", EventID: 3, WorkingFrom: today.Add(10 * time.Hour), WorkingTo: today.Add(18 * time.Hour)},
		{UserID: 3, UserName: "Charlie", EventID: 2, WorkingFrom: today.Add(8 * time.Hour), WorkingTo: today.Add(16 * time.Hour)},
		{UserID: 3, UserName: "Charlie", EventID: 3, WorkingFrom: today.Add(8 * time.Hour), WorkingTo: today.Add(16 * time.Hour)},
		// Add schedules for conflicting events
		{UserID: 1, UserName: "Alice", EventID: 5, WorkingFrom: today.Add(9 * time.Hour), WorkingTo: today.Add(17 * time.Hour)}, // Alice in both Team Meeting and Marketing Standup
		{UserID: 2, UserName: "Bob", EventID: 6, WorkingFrom: today.Add(10 * time.Hour), WorkingTo: today.Add(18 * time.Hour)},   // Bob in both Client Presentation and Board Meeting
	}

	for _, s := range schedules {
		db.MustExec(
			"INSERT INTO user_schedules (user_id, user_name, event_id, working_from, working_to) VALUES (?, ?, ?, ?, ?)",
			s.UserID, s.UserName, s.EventID, s.WorkingFrom, s.WorkingTo,
		)
	}

	log.Println("Database initialized with sample data")
}