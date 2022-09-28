package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/DDexster/go-vigilate/internal/helpers"
	"github.com/DDexster/go-vigilate/internal/models"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	HTTP           = 1
	HTTPS          = 2
	SSLCertificate = 3
)

type jsonResponse struct {
	OK            bool      `json:"ok"`
	Message       string    `json:"message"`
	ServiceID     int       `json:"service_id"`
	HostServiceID int       `json:"host_service_id"`
	HostId        int       `json:"host_id"`
	OldStatus     string    `json:"old_status"`
	NewStatus     string    `json:"new_status"`
	LastCheck     time.Time `json:"last_check"`
}

func (repo *DBRepo) ScheduledCheck(hsID int) {
	log.Println(fmt.Sprintf("***** Running check for %d host service", hsID))

	hs, err := repo.DB.GetHostServiceById(hsID)
	if err != nil {
		log.Println(err)
		return
	}

	h, err := repo.DB.GetHostById(hs.HostID)
	if err != nil {
		log.Println(err)
		return
	}

	newStatus, msg := repo.testServiceHost(h, hs)

	// update hs time check and status
	statusChanged := hs.Status != newStatus
	hs.Status = newStatus
	hs.LastCheck = time.Now()

	err = repo.DB.UpdateHostService(hs)
	if err != nil {
		log.Println(err)
		return
	}

	if statusChanged {
		updateMessage := fmt.Sprintf("Host service %s on %s is changed to %s, message: %s", hs.Service.ServiceName, h.HostName, newStatus, msg)
		repo.updateServiceStatusCount(updateMessage)
		//	TODO alert user via email or sms
	}
}

func (repo *DBRepo) TestCheck(w http.ResponseWriter, r *http.Request) {
	hostServiceID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	oldStatus := chi.URLParam(r, "oldStatus")
	okay := true

	// get hs
	hs, err := repo.DB.GetHostServiceById(hostServiceID)
	if err != nil {
		log.Println(err)
		okay = false
	}

	// get host?
	h, err := repo.DB.GetHostById(hs.HostID)
	if err != nil {
		log.Println(err)
		okay = false
	}

	// test service
	newStatus, msg := repo.testServiceHost(h, hs)

	// update hs time check and status
	hs.Status = newStatus
	hs.LastCheck = time.Now()

	err = repo.DB.UpdateHostService(hs)
	if err != nil {
		log.Println(err)
		okay = false
	}

	// broadcast service status change

	var response jsonResponse

	if okay {
		response = jsonResponse{
			OK:            okay,
			Message:       msg,
			NewStatus:     newStatus,
			OldStatus:     oldStatus,
			ServiceID:     hs.ServiceID,
			HostServiceID: hs.ID,
			HostId:        hs.HostID,
			LastCheck:     hs.LastCheck,
		}
	} else {
		response = jsonResponse{
			OK:      okay,
			Message: "Ooops,  something went wrong",
		}
	}

	//	create response and send back
	out, _ := json.MarshalIndent(response, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (repo *DBRepo) testServiceHost(h models.Host, hs models.HostService) (string, string) {
	var msg, newStatus string

	switch hs.ServiceID {
	case HTTP:
		msg, newStatus = testHTTPForHost(h.URL)
		break
	}

	if hs.Status != newStatus {
		repo.pushHostServiceStatusChange(hs, newStatus)

		// TODO send email or sms if appropriate
	}

	// broadcast schedule-changed-event
	repo.pushHostServiceScheduleChange(hs, newStatus)

	return newStatus, msg
}

func testHTTPForHost(url string) (string, string) {
	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}

	url = strings.Replace(url, "https://", "http://", -1)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("%s - %s", url, "error connecting"), "problem"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("%s - %s", url, resp.Status), "problem"
	}

	return fmt.Sprintf("%s - %s", url, resp.Status), "healthy"
}

func (repo *DBRepo) broadcastMessage(channel, event string, data map[string]string) {
	err := app.WsClient.Trigger(channel, event, data)
	if err != nil {
		log.Println(err)
	}
}

func (repo *DBRepo) updateServiceStatusCount(message string) {
	log.Println(message)

	//	broadcast to all clients
	data := make(map[string]string)
	counts := repo.GetServiceCounts(nil)
	data["message"] = fmt.Sprintf(message)
	data["healthy_count"] = strconv.Itoa(counts.Healthy)
	data["pending_count"] = strconv.Itoa(counts.Pending)
	data["problem_count"] = strconv.Itoa(counts.Problem)
	data["warning_count"] = strconv.Itoa(counts.Warning)
	repo.broadcastMessage("public-channel", "hs-count-changed", data)
}

func (repo *DBRepo) pushHostServiceStatusChange(hs models.HostService, newStatus string) {
	data := make(map[string]string)
	data["host_service_id"] = strconv.Itoa(hs.ID)
	data["host_id"] = strconv.Itoa(hs.HostID)
	data["service_id"] = strconv.Itoa(hs.ServiceID)
	data["host_name"] = hs.HostName
	data["service_name"] = hs.Service.ServiceName
	data["icon"] = hs.Service.Icon
	data["status"] = newStatus
	data["last_check"] = helpers.FormatDateWithLayout(time.Now(), helpers.DATE_FORMAT)
	data["message"] = fmt.Sprintf("%s on %s reports %s", hs.Service.ServiceName, hs.HostName, newStatus)

	repo.broadcastMessage("public-channel", "host-service-status-change", data)
}

func (repo *DBRepo) pushHostServiceScheduleChange(hs models.HostService, newStatus string) {
	yearOne := time.Date(0001, 1, 1, 0, 0, 0, 1, time.UTC)
	data := make(map[string]string)
	data["host_service_id"] = strconv.Itoa(hs.ID)
	data["service_id"] = strconv.Itoa(hs.ServiceID)
	data["host_id"] = strconv.Itoa(hs.HostID)

	if app.Scheduler.Entry(repo.App.MonitorMap[hs.ID]).Next.After(yearOne) {
		data["next_run"] = repo.App.Scheduler.Entry(repo.App.MonitorMap[hs.ID]).Next.Format(helpers.DATE_SCHEDULE_FORMAT)
	} else {
		data["next_run"] = "Pending..."
	}

	data["last_check"] = time.Now().Format(helpers.DATE_SCHEDULE_FORMAT)
	data["host_name"] = hs.HostName
	data["service_name"] = hs.Service.ServiceName
	data["schedule"] = fmt.Sprintf("@every %d%s", hs.ScheduleNumber, hs.ScheduleUnit)
	data["status"] = newStatus
	data["icon"] = hs.Service.Icon
	//repo.broadcastMessage("public-channel", "next-run-event", data)

	repo.broadcastMessage("public-channel", "schedule-changed-event", data)
}

func (repo *DBRepo) pushHostServiceScheduleRemoved(hs models.HostService) {
	data := make(map[string]string)

	data["host_service_id"] = strconv.Itoa(hs.ID)
	//repo.broadcastMessage("public-channel", "next-run-event", data)

	repo.broadcastMessage("public-channel", "schedule-removed-event", data)
}

func (repo *DBRepo) addHostServiceToMonitorMap(hs models.HostService) {
	if repo.App.PreferenceMap["monitoring_live"] == "1" {
		log.Println(fmt.Sprintf("*** Service to monitor on %s is %s", hs.HostName, hs.Service.ServiceName))
		var j job
		j.HostServiceId = hs.ID
		sch := fmt.Sprintf("@every %d%s", hs.ScheduleNumber, hs.ScheduleUnit)
		if hs.ScheduleUnit == "d" {
			sch = fmt.Sprintf("@every %d%s", hs.ScheduleNumber*24, "h")
		}
		scheduleID, err := repo.App.Scheduler.AddJob(sch, j)
		if err != nil {
			log.Println(err)
			return
		}
		repo.App.MonitorMap[hs.ID] = scheduleID
		repo.pushHostServiceScheduleChange(hs, "pending")
	}
}

func (repo *DBRepo) removeHostServiceFromMonitorMap(hs models.HostService) {
	if repo.App.PreferenceMap["monitoring_live"] == "1" {
		schID, ok := repo.App.MonitorMap[hs.ID]
		if !ok {
			log.Println("Error removing from monitor service")
			return
		}

		repo.App.Scheduler.Remove(schID)
		delete(repo.App.MonitorMap, hs.ID)
		repo.pushHostServiceScheduleRemoved(hs)
	}
}
