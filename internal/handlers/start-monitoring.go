package handlers

import (
	"fmt"
	"github.com/DDexster/go-vigilate/internal/helpers"
	"log"
	"strconv"
	"time"
)

type job struct {
	HostServiceId int
}

func (j job) Run() {
	Repo.ScheduledCheck(j.HostServiceId)
}

func (repo *DBRepo) StartMonitoring() {
	if app.PreferenceMap["monitoring_live"] == "1" {
		log.Println("********* Starting Schedule Monitoring *********")
		data := make(map[string]string)
		data["message"] = "Monitoring is starting..."

		// trigger message to broadcast to all clients
		repo.broadcastMessage("public-channel", "app-started", data)

		// get all services that we want to monitor
		servicesToMonitor, err := repo.DB.GetServicesToMonitor()
		if err != nil {
			log.Println(err)
		}

		// range via services and define schedule
		for _, hs := range servicesToMonitor {
			log.Println(fmt.Sprintf("*** Service to monitor on %s is %s", hs.HostName, hs.Service.ServiceName))
			sch := fmt.Sprintf("@every %d%s", hs.ScheduleNumber, hs.ScheduleUnit)
			if hs.ScheduleUnit == "d" {
				sch = fmt.Sprintf("@every %d%s", hs.ScheduleNumber*24, "h")
			}

			// create job
			j := job{
				HostServiceId: hs.ID,
			}
			schID, err := app.Scheduler.AddJob(sch, j)
			if err != nil {
				log.Println(err)
			}

			// save id of job to control it
			app.MonitorMap[hs.ID] = schID

			// broadcast over websockets that schedule is started
			payload := make(map[string]string)
			payload["message"] = "scheduling"
			payload["host_service_id"] = strconv.Itoa(hs.ID)
			yearOne := time.Date(001, 11, 17, 23, 32, 23, 432322, time.UTC)
			if app.Scheduler.Entry(app.MonitorMap[hs.ID]).Next.After(yearOne) {
				payload["next_run"] = app.Scheduler.Entry(app.MonitorMap[hs.ID]).Next.Format(helpers.DATE_FORMAT)
			} else {
				payload["next_run"] = "Pending..."
			}
			payload["host"] = hs.HostName
			payload["service"] = hs.Service.ServiceName
			if hs.LastCheck.After(yearOne) {
				payload["last_run"] = hs.LastCheck.Format(helpers.DATE_FORMAT)
			} else {
				payload["last_run"] = "Pending..."
			}
			payload["schedule"] = sch

			repo.broadcastMessage("public-channel", "next-run-event", payload)
			repo.broadcastMessage("public-channel", "schedule-changed-event", payload)
		}
	}
}
