package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/CloudyKit/jet/v6"
	"github.com/DDexster/go-vigilate/internal/config"
	"github.com/DDexster/go-vigilate/internal/driver"
	"github.com/DDexster/go-vigilate/internal/helpers"
	"github.com/DDexster/go-vigilate/internal/models"
	"github.com/DDexster/go-vigilate/internal/repository"
	"github.com/DDexster/go-vigilate/internal/repository/dbrepo"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
)

// Repo is the repository
var Repo *DBRepo
var app *config.AppConfig

// DBRepo is the db repo
type DBRepo struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

// NewHandlers creates the handlers
func NewHandlers(repo *DBRepo, a *config.AppConfig) {
	Repo = repo
	app = a
}

// NewPostgresqlHandlers creates db repo for postgres
func NewPostgresqlHandlers(db *driver.DB, a *config.AppConfig) *DBRepo {
	return &DBRepo{
		App: a,
		DB:  dbrepo.NewPostgresRepo(db.SQL, a),
	}
}

func (repo *DBRepo) GetServiceCounts(hosts *[]models.Host) models.ServiceStatusCount {
	var counts models.ServiceStatusCount
	if hosts != nil {
		for _, h := range *hosts {
			if h.Active == 1 {
				for _, hs := range h.HostServices {
					if hs.Active == 1 {
						switch hs.Status {
						case "pending":
							counts.Pending += 1
						case "healthy":
							counts.Healthy += 1
						case "problem":
							counts.Problem += 1
						case "warning":
							counts.Warning += 1
						}
					}
				}
			}
		}
	} else {
		cts, err := repo.DB.GetAllServiceStatusCounts()
		if err != nil {
			log.Println(err)
			return counts
		}
		counts = cts
	}
	return counts
}

// AdminDashboard displays the dashboard
func (repo *DBRepo) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	hosts, err := repo.DB.GetAllHosts()
	if err != nil {
		log.Println(err)
		return
	}

	counts := repo.GetServiceCounts(&hosts)

	vars := make(jet.VarMap)
	vars.Set("hosts", hosts)
	vars.Set("no_healthy", counts.Healthy)
	vars.Set("no_problem", counts.Problem)
	vars.Set("no_pending", counts.Pending)
	vars.Set("no_warning", counts.Warning)

	err = helpers.RenderPage(w, r, "dashboard", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// Events displays the events page
func (repo *DBRepo) Events(w http.ResponseWriter, r *http.Request) {
	events, err := repo.DB.GetAllEvents()
	if err != nil {
		log.Println(err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("events", events)

	err = helpers.RenderPage(w, r, "events", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// Settings displays the settings page
func (repo *DBRepo) Settings(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "settings", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// PostSettings saves site settings
func (repo *DBRepo) PostSettings(w http.ResponseWriter, r *http.Request) {
	prefMap := make(map[string]string)

	prefMap["site_url"] = r.Form.Get("site_url")
	prefMap["notify_name"] = r.Form.Get("notify_name")
	prefMap["notify_email"] = r.Form.Get("notify_email")
	prefMap["smtp_server"] = r.Form.Get("smtp_server")
	prefMap["smtp_port"] = r.Form.Get("smtp_port")
	prefMap["smtp_user"] = r.Form.Get("smtp_user")
	prefMap["smtp_password"] = r.Form.Get("smtp_password")
	prefMap["sms_enabled"] = r.Form.Get("sms_enabled")
	prefMap["sms_provider"] = r.Form.Get("sms_provider")
	prefMap["twilio_phone_number"] = r.Form.Get("twilio_phone_number")
	prefMap["twilio_sid"] = r.Form.Get("twilio_sid")
	prefMap["twilio_auth_token"] = r.Form.Get("twilio_auth_token")
	prefMap["smtp_from_email"] = r.Form.Get("smtp_from_email")
	prefMap["smtp_from_name"] = r.Form.Get("smtp_from_name")
	prefMap["notify_via_sms"] = r.Form.Get("notify_via_sms")
	prefMap["notify_via_email"] = r.Form.Get("notify_via_email")
	prefMap["sms_notify_number"] = r.Form.Get("sms_notify_number")

	if r.Form.Get("sms_enabled") == "0" {
		prefMap["notify_via_sms"] = "0"
	}

	err := repo.DB.InsertOrUpdateSitePreferences(prefMap)
	if err != nil {
		log.Println(err)
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	// update app config
	for k, v := range prefMap {
		app.PreferenceMap[k] = v
	}

	app.Session.Put(r.Context(), "flash", "Changes saved")

	if r.Form.Get("action") == "1" {
		http.Redirect(w, r, "/admin/overview", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
	}
}

func (repo *DBRepo) UpdateSystemPreference(w http.ResponseWriter, r *http.Request) {
	resp := jsonResponse{
		OK:      false,
		Message: "Failed to update system preference",
	}
	w.Header().Set("Content-Type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		resp.Message = "Failed to parse form" + err.Error()
		out, _ := json.MarshalIndent(resp, "", " ")
		w.Write(out)
		return
	}

	prefName := r.Form.Get("pref_name")
	prefValue := r.Form.Get("pref_value")

	err = repo.DB.UpdateSystemPref(prefName, prefValue)
	if err != nil {
		resp.Message = "Failed to update preference" + err.Error()
		out, _ := json.MarshalIndent(resp, "", " ")
		w.Write(out)
		return
	}
	repo.App.PreferenceMap[prefName] = prefValue

	resp.OK = true
	resp.Message = "Updated Successfully"
	out, _ := json.MarshalIndent(resp, "", " ")
	w.Write(out)
}

func (repo *DBRepo) ToggleMonitoring(w http.ResponseWriter, r *http.Request) {
	active := r.Form.Get("active")

	if active == "1" {
		log.Println("Turning Monitoring on...")
		repo.App.PreferenceMap["monitoring_live"] = "1"
		repo.StartMonitoring()
		repo.App.Scheduler.Start()
	} else {
		log.Println("Turning Monitoring off...")
		repo.App.PreferenceMap["monitoring_live"] = "0"
		for _, v := range repo.App.MonitorMap {
			repo.App.Scheduler.Remove(v)
		}
		for k := range repo.App.MonitorMap {
			delete(repo.App.MonitorMap, k)
		}
		// delete all entries from scheduler
		for _, i := range repo.App.Scheduler.Entries() {
			repo.App.Scheduler.Remove(i.ID)
		}
		repo.App.Scheduler.Stop()
		data := make(map[string]string)
		data["message"] = "Monitoring is stopped"

		// trigger message to broadcast to all clients
		err := app.WsClient.Trigger("public-channel", "app-stopped", data)
		if err != nil {
			log.Println(err)
		}
	}

	resp := jsonResponse{
		OK: true,
	}

	out, _ := json.MarshalIndent(resp, "", " ")
	w.Write(out)
}

// AllHosts displays list of all hosts
func (repo *DBRepo) AllHosts(w http.ResponseWriter, r *http.Request) {
	hosts, err := repo.DB.GetAllHosts()

	if err != nil {
		log.Println(err)
	}

	vars := make(jet.VarMap)
	vars.Set("hosts", hosts)

	err = helpers.RenderPage(w, r, "hosts", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// Host shows the host add/edit form
func (repo *DBRepo) Host(w http.ResponseWriter, r *http.Request) {
	hostID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var h models.Host
	if hostID > 0 {
		//	get host from db
		hst, err := repo.DB.GetHostById(hostID)
		if err != nil {
			log.Println(err)
			repo.App.Session.Put(r.Context(), "error", "Host Not Found")
			http.Redirect(w, r, "/admin/host/all", http.StatusSeeOther)
			return
		}
		h = hst
	}

	vars := make(jet.VarMap)
	vars.Set("host", h)

	err := helpers.RenderPage(w, r, "host", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// PostHost handles submit host add/edit form
func (repo *DBRepo) PostHost(w http.ResponseWriter, r *http.Request) {
	hostID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var newHost models.Host

	if hostID > 0 {
		h, err := repo.DB.GetHostById(hostID)
		if err != nil {
			log.Println(err)
			helpers.ServerError(w, r, err)
			return
		}

		newHost = h
	}

	newHost.HostName = r.Form.Get("host_name")
	newHost.CanonicalName = r.Form.Get("canonical_name")
	newHost.URL = r.Form.Get("url")
	newHost.IP = r.Form.Get("ip")
	newHost.IPV6 = r.Form.Get("ipv6")
	newHost.Location = r.Form.Get("location")
	newHost.OS = r.Form.Get("os")
	active, _ := strconv.Atoi(r.Form.Get("active"))
	newHost.Active = active

	log.Println("host id", hostID)
	log.Println("host active", active)

	if hostID > 0 {
		err := repo.DB.UpdateHost(newHost)
		if err != nil {
			log.Println(err)
			helpers.ServerError(w, r, err)
			return
		}
		//	get host from db
	} else {
		newId, err := repo.DB.InsertHost(newHost)
		if err != nil {
			log.Println(err)
			helpers.ServerError(w, r, err)
			return
		}
		newHost.ID = newId
	}

	repo.App.Session.Put(r.Context(), "flash", "Changes Saved")
	action := r.Form.Get("action")
	redirectUrl := fmt.Sprintf("/admin/host/%d", newHost.ID)
	if action == "1" {
		redirectUrl = "/admin/host/all"
	}
	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
}

type serviceJson struct {
	OK bool `json:"ok"`
}

func (repo *DBRepo) ToggleHostService(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}
	hostId, _ := strconv.Atoi(r.Form.Get("host_id"))
	serviceId, _ := strconv.Atoi(r.Form.Get("service_id"))
	active, _ := strconv.Atoi(r.Form.Get("active"))

	response := serviceJson{
		OK: true,
	}

	err = repo.DB.UpdateHostServiceStatus(hostId, serviceId, active)
	if err != nil {
		log.Println(err)
		response.OK = false
	}

	//broadcast
	hs, _ := repo.DB.GetHostServiceByHostIDServiceID(hostId, serviceId)

	// add or remove schedule for service
	if active == 1 {
		repo.pushHostServiceStatusChange(hs, "pending")
		repo.pushHostServiceScheduleChange(hs, "pending")
		repo.addHostServiceToMonitorMap(hs)
	} else {
		repo.removeHostServiceFromMonitorMap(hs)
	}

	out, _ := json.MarshalIndent(response, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// AllUsers lists all admin users
func (repo *DBRepo) AllUsers(w http.ResponseWriter, r *http.Request) {
	vars := make(jet.VarMap)

	u, err := repo.DB.AllUsers()
	if err != nil {
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	vars.Set("users", u)

	err = helpers.RenderPage(w, r, "users", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// OneUser displays the add/edit user page
func (repo *DBRepo) OneUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Println(err)
	}

	vars := make(jet.VarMap)

	if id > 0 {

		u, err := repo.DB.GetUserById(id)
		if err != nil {
			ClientError(w, r, http.StatusBadRequest)
			return
		}

		vars.Set("user", u)
	} else {
		var u models.User
		vars.Set("user", u)
	}

	err = helpers.RenderPage(w, r, "user", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// PostOneUser adds/edits a user
func (repo *DBRepo) PostOneUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Println(err)
	}

	var u models.User

	if id > 0 {
		u, _ = repo.DB.GetUserById(id)
		u.FirstName = r.Form.Get("first_name")
		u.LastName = r.Form.Get("last_name")
		u.Email = r.Form.Get("email")
		u.UserActive, _ = strconv.Atoi(r.Form.Get("user_active"))
		err := repo.DB.UpdateUser(u)
		if err != nil {
			log.Println(err)
			ClientError(w, r, http.StatusBadRequest)
			return
		}

		if len(r.Form.Get("password")) > 0 {
			// changing password
			err := repo.DB.UpdatePassword(id, r.Form.Get("password"))
			if err != nil {
				log.Println(err)
				ClientError(w, r, http.StatusBadRequest)
				return
			}
		}
	} else {
		u.FirstName = r.Form.Get("first_name")
		u.LastName = r.Form.Get("last_name")
		u.Email = r.Form.Get("email")
		u.UserActive, _ = strconv.Atoi(r.Form.Get("user_active"))
		u.Password = []byte(r.Form.Get("password"))
		u.AccessLevel = 3

		_, err := repo.DB.InsertUser(u)
		if err != nil {
			log.Println(err)
			ClientError(w, r, http.StatusBadRequest)
			return
		}
	}

	repo.App.Session.Put(r.Context(), "flash", "Changes saved")
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// DeleteUser soft deletes a user
func (repo *DBRepo) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	_ = repo.DB.DeleteUser(id)
	repo.App.Session.Put(r.Context(), "flash", "User deleted")
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// ClientError will display error page for client error i.e. bad request
func ClientError(w http.ResponseWriter, r *http.Request, status int) {
	switch status {
	case http.StatusNotFound:
		show404(w, r)
	case http.StatusInternalServerError:
		show500(w, r)
	default:
		http.Error(w, http.StatusText(status), status)
	}
}

// ServerError will display error page for internal server error
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	_ = log.Output(2, trace)
	show500(w, r)
}

func show404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	http.ServeFile(w, r, "./ui/static/404.html")
}

func show500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	http.ServeFile(w, r, "./ui/static/500.html")
}

func printTemplateError(w http.ResponseWriter, err error) {
	_, _ = fmt.Fprint(w, fmt.Sprintf(`<small><span class='text-danger'>Error executing template: %s</span></small>`, err))
}
