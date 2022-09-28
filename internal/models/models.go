package models

import (
	"errors"
	"github.com/robfig/cron/v3"
	"time"
)

var (
	// ErrNoRecord no record found in database error
	ErrNoRecord = errors.New("models: no matching record found")
	// ErrInvalidCredentials invalid username/password error
	ErrInvalidCredentials = errors.New("models: invalid credentials")
	// ErrDuplicateEmail duplicate email error
	ErrDuplicateEmail = errors.New("models: duplicate email")
	// ErrInactiveAccount inactive account error
	ErrInactiveAccount = errors.New("models: Inactive Account")
)

type ServiceStatusCount struct {
	Healthy int
	Problem int
	Warning int
	Pending int
}

// User model
type User struct {
	ID          int
	FirstName   string
	LastName    string
	UserActive  int
	AccessLevel int
	Email       string
	Password    []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
	Preferences map[string]string
}

// Preference model
type Preference struct {
	ID         int
	Name       string
	Preference []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Host model
type Host struct {
	ID            int
	HostName      string
	CanonicalName string
	URL           string
	IP            string
	IPV6          string
	Location      string
	OS            string
	Active        int
	HostServices  []HostService
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Service model
type Service struct {
	ID          int
	ServiceName string
	Active      int
	Icon        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// HostService model
type HostService struct {
	ID             int
	ServiceID      int
	HostID         int
	Active         int
	ScheduleNumber int
	ScheduleUnit   string
	Status         string
	LastCheck      time.Time
	LastMessage    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Service        Service
	HostName       string
}

type Schedule struct {
	ID            int
	EntryID       cron.EntryID
	Entry         cron.Entry
	Host          string
	Service       string
	LastRunFromHS time.Time
	HostServiceID int
	ScheduleText  string
}

type Event struct {
	ID            int
	EventType     string
	HostServiceID int
	HostID        int
	ServiceName   string
	HostName      string
	Message       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
