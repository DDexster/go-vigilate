package handlers

import (
	"log"
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
			repo.addHostServiceToMonitorMap(hs)
			repo.pushHostServiceScheduleChange(hs, "pending")
		}
	}
}
