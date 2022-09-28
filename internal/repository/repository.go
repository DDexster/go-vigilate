package repository

import (
	"github.com/DDexster/go-vigilate/internal/models"
)

// DatabaseRepo is the database repository
type DatabaseRepo interface {
	// preferences
	AllPreferences() ([]models.Preference, error)
	SetSystemPref(name, value string) error
	UpdateSystemPref(name, value string) error
	InsertOrUpdateSitePreferences(pm map[string]string) error

	// users and authentication
	GetUserById(id int) (models.User, error)
	InsertUser(u models.User) (int, error)
	UpdateUser(u models.User) error
	DeleteUser(id int) error
	UpdatePassword(id int, newPassword string) error
	Authenticate(email, testPassword string) (int, string, error)
	AllUsers() ([]*models.User, error)
	InsertRememberMeToken(id int, token string) error
	DeleteToken(token string) error
	CheckForToken(id int, token string) bool

	// hosts
	GetAllHosts() ([]models.Host, error)
	GetHostById(id int) (models.Host, error)
	InsertHost(h models.Host) (int, error)
	UpdateHost(h models.Host) error
	UpdateHostServiceStatus(hostId, serviceId, active int) error
	GetHostServiceByHostIDServiceID(hostId, serviceId int) (models.HostService, error)
	GetHostServicesByStatus(status string) ([]models.HostService, error)
	GetHostServiceById(id int) (models.HostService, error)
	UpdateHostService(hs models.HostService) error
	GetServicesToMonitor() ([]models.HostService, error)
	GetAllServiceStatusCounts() (models.ServiceStatusCount, error)
}
