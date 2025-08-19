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
	ID          int    `db:"id"`
	UserID      int    `db:"user_id"`
	UserName    string `db:"user_name"`
	EventID     int    `db:"event_id"`
	WorkingFrom string `db:"working_from"`
	WorkingTo   string `db:"working_to"`
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
		working_from TEXT NOT NULL,
		working_to TEXT NOT NULL,
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
	}

	for _, e := range events {
		db.MustExec(
			"INSERT INTO events (name, date, location, duration_hours) VALUES (?, ?, ?, ?)",
			e.Name, e.Date, e.Location, e.Duration,
		)
	}

	schedules := []UserSchedule{
		{UserID: 1, UserName: "Alice", EventID: 1, WorkingFrom: "09:00", WorkingTo: "17:00"},
		{UserID: 1, UserName: "Alice", EventID: 2, WorkingFrom: "09:00", WorkingTo: "17:00"},
		{UserID: 2, UserName: "Bob", EventID: 1, WorkingFrom: "10:00", WorkingTo: "18:00"},
		{UserID: 2, UserName: "Bob", EventID: 3, WorkingFrom: "10:00", WorkingTo: "18:00"},
		{UserID: 3, UserName: "Charlie", EventID: 2, WorkingFrom: "08:00", WorkingTo: "16:00"},
		{UserID: 3, UserName: "Charlie", EventID: 3, WorkingFrom: "08:00", WorkingTo: "16:00"},
	}

	for _, s := range schedules {
		db.MustExec(
			"INSERT INTO user_schedules (user_id, user_name, event_id, working_from, working_to) VALUES (?, ?, ?, ?, ?)",
			s.UserID, s.UserName, s.EventID, s.WorkingFrom, s.WorkingTo,
		)
	}

	log.Println("Database initialized with sample data")
}