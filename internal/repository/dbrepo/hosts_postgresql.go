package dbrepo

import (
	"context"
	"github.com/DDexster/go-vigilate/internal/models"
	"log"
	"time"
)

// GetAllHosts returns a slice of hosts
func (m *postgresDBRepo) GetAllHosts() ([]models.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT id, host_name, canonical_name, url, ip, ipv6, location, os, active, created_at, updated_at 
	FROM hosts 
	ORDER BY host_name`

	rows, err := m.DB.QueryContext(ctx, stmt)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	var hosts []models.Host

	for rows.Next() {
		h := &models.Host{}
		err = rows.Scan(
			&h.ID,
			&h.HostName,
			&h.CanonicalName,
			&h.URL,
			&h.IP,
			&h.IPV6,
			&h.Location,
			&h.OS,
			&h.Active,
			&h.CreatedAt,
			&h.UpdatedAt,
		)

		if err != nil {
			log.Println(err)
			return nil, err
		}

		serviceQuery := `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.last_message, hs.status, hs.created_at, hs.updated_at, 
       s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at
		FROM host_services AS hs
		LEFT JOIN services AS s ON (hs.service_id = s.id)
		WHERE hs.host_id = $1`

		serviceRows, err := m.DB.QueryContext(ctx, serviceQuery, h.ID)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		var hostServices []models.HostService

		for serviceRows.Next() {
			var hs models.HostService
			err = serviceRows.Scan(
				&hs.ID,
				&hs.HostID,
				&hs.ServiceID,
				&hs.Active,
				&hs.ScheduleNumber,
				&hs.ScheduleUnit,
				&hs.LastCheck,
				&hs.LastMessage,
				&hs.Status,
				&hs.CreatedAt,
				&hs.UpdatedAt,
				&hs.Service.ID,
				&hs.Service.ServiceName,
				&hs.Service.Active,
				&hs.Service.Icon,
				&hs.Service.CreatedAt,
				&hs.Service.UpdatedAt,
			)

			if err != nil {
				log.Println(err)
				return nil, err
			}

			hostServices = append(hostServices, hs)
		}

		h.HostServices = hostServices

		hosts = append(hosts, *h)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return hosts, nil
}

func (m *postgresDBRepo) InsertHost(h models.Host) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO hosts 
	    (host_name, canonical_name, url, ip, ipv6, location, os, active, created_at, updated_at)
    VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`

	var newId int
	err := m.DB.QueryRowContext(ctx, stmt,
		h.HostName,
		h.CanonicalName,
		h.URL,
		h.IP,
		h.IPV6,
		h.Location,
		h.OS,
		h.Active,
		time.Now(),
		time.Now(),
	).Scan(&newId)

	if err != nil {
		log.Println(err)
		return 0, err
	}

	// Add Host Services and set to inactive
	stmt = `SELECT id FROM services`
	serviceRows, err := m.DB.QueryContext(ctx, stmt)
	if err != nil {
		log.Println(err)
		return newId, err
	}
	defer serviceRows.Close()

	for serviceRows.Next() {
		var sID int
		err := serviceRows.Scan(&sID)
		if err != nil {
			log.Println(err)
			return newId, err
		}

		stmt = `INSERT INTO host_services (host_id, service_id, active, schedule_number, schedule_unit, status, created_at, updated_at) 
		VALUES ($1, $2, 0, 1, 'm', 'pending', $3, $4)`

		_, err = m.DB.ExecContext(ctx, stmt,
			newId,
			sID,
			time.Now(),
			time.Now(),
		)

		if err != nil {
			log.Println(err)
			return newId, err
		}
	}

	return newId, nil
}

func (m *postgresDBRepo) GetHostById(id int) (models.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT 
    	id, host_name, canonical_name, url, ip, ipv6, location, os, active, created_at, updated_at
		FROM hosts WHERE id = $1`

	var h models.Host
	row := m.DB.QueryRowContext(ctx, stmt, id)

	err := row.Scan(
		&h.ID,
		&h.HostName,
		&h.CanonicalName,
		&h.URL,
		&h.IP,
		&h.IPV6,
		&h.Location,
		&h.OS,
		&h.Active,
		&h.CreatedAt,
		&h.UpdatedAt,
	)

	if err != nil {
		log.Println(err)
		return h, err
	}

	//get all host hostServices
	stmt = `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.last_message, hs.status, hs.created_at, hs.updated_at, 
       s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at
		FROM host_services AS hs
		LEFT JOIN services AS s ON (hs.service_id = s.id)
		WHERE hs.host_id = $1
		ORDER BY s.service_name`

	rows, err := m.DB.QueryContext(ctx, stmt, h.ID)
	if err != nil {
		log.Println(err)
		return h, err
	}
	defer rows.Close()

	var hostServices []models.HostService

	for rows.Next() {
		var hs models.HostService
		err := rows.Scan(
			&hs.ID,
			&hs.HostID,
			&hs.ServiceID,
			&hs.Active,
			&hs.ScheduleNumber,
			&hs.ScheduleUnit,
			&hs.LastCheck,
			&hs.LastMessage,
			&hs.Status,
			&hs.CreatedAt,
			&hs.UpdatedAt,
			&hs.Service.ID,
			&hs.Service.ServiceName,
			&hs.Service.Active,
			&hs.Service.Icon,
			&hs.Service.CreatedAt,
			&hs.Service.UpdatedAt,
		)
		if err != nil {
			log.Println(err)
			return h, err
		}
		hostServices = append(hostServices, hs)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return h, err
	}

	h.HostServices = hostServices

	return h, nil
}

func (m *postgresDBRepo) UpdateHost(h models.Host) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE hosts
		SET 
		    host_name = $1,
		    canonical_name = $2, 
		    url = $3, 
		    ip = $4, 
		    ipv6 = $5,
		    location = $6,
		    os = $7,
		    active = $8,
		    updated_at = $9
		WHERE id = $10`

	_, err := m.DB.ExecContext(ctx, stmt,
		h.HostName,
		h.CanonicalName,
		h.URL,
		h.IP,
		h.IPV6,
		h.Location,
		h.OS,
		h.Active,
		time.Now(),
		h.ID,
	)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (m *postgresDBRepo) UpdateHostServiceStatus(hostId, serviceId, active int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE host_services SET active = $1 WHERE host_id = $2 AND service_id = $3`

	_, err := m.DB.ExecContext(ctx, stmt, active, hostId, serviceId)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (m *postgresDBRepo) GetHostServicesByStatus(status string) ([]models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.last_message, hs.status, hs.created_at, hs.updated_at, 
       h.host_name, s.service_name
		FROM host_services AS hs
		LEFT JOIN hosts AS h ON (hs.host_id = h.id)
		LEFT JOIN services AS s ON (hs.service_id = s.id)
		WHERE hs.status = $1 AND hs.active = 1
		ORDER BY h.host_name, s.service_name`

	rows, err := m.DB.QueryContext(ctx, stmt, status)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	var services []models.HostService

	for rows.Next() {
		var hs models.HostService
		err := rows.Scan(
			&hs.ID,
			&hs.HostID,
			&hs.ServiceID,
			&hs.Active,
			&hs.ScheduleNumber,
			&hs.ScheduleUnit,
			&hs.LastCheck,
			&hs.LastMessage,
			&hs.Status,
			&hs.CreatedAt,
			&hs.UpdatedAt,
			&hs.HostName,
			&hs.Service.ServiceName,
		)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		services = append(services, hs)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return services, err
	}

	return services, nil
}

func (m *postgresDBRepo) GetHostServiceById(id int) (models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.last_message, hs.status, hs.created_at, hs.updated_at, 
      s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at,
			h.host_name
		FROM host_services AS hs
		LEFT JOIN services AS s ON (hs.service_id = s.id)
		LEFT JOIN hosts AS h ON (hs.host_id = h.id)
		WHERE hs.id = $1`

	var hs models.HostService

	row := m.DB.QueryRowContext(ctx, stmt, id)
	err := row.Scan(
		&hs.ID,
		&hs.HostID,
		&hs.ServiceID,
		&hs.Active,
		&hs.ScheduleNumber,
		&hs.ScheduleUnit,
		&hs.LastCheck,
		&hs.LastMessage,
		&hs.Status,
		&hs.CreatedAt,
		&hs.UpdatedAt,
		&hs.Service.ID,
		&hs.Service.ServiceName,
		&hs.Service.Active,
		&hs.Service.Icon,
		&hs.Service.CreatedAt,
		&hs.Service.UpdatedAt,
		&hs.HostName,
	)

	if err != nil {
		log.Println(err)
		return hs, err
	}

	return hs, nil
}

func (m *postgresDBRepo) GetHostServiceByHostIDServiceID(hostId, serviceId int) (models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.last_message, hs.status, hs.created_at, hs.updated_at, 
      s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at,
			h.host_name
		FROM host_services AS hs
		LEFT JOIN services AS s ON (hs.service_id = s.id)
		LEFT JOIN hosts AS h ON (hs.host_id = h.id)
		WHERE hs.host_id = $1 AND hs.service_id = $2`

	var hs models.HostService

	row := m.DB.QueryRowContext(ctx, stmt, hostId, serviceId)
	err := row.Scan(
		&hs.ID,
		&hs.HostID,
		&hs.ServiceID,
		&hs.Active,
		&hs.ScheduleNumber,
		&hs.ScheduleUnit,
		&hs.LastCheck,
		&hs.LastMessage,
		&hs.Status,
		&hs.CreatedAt,
		&hs.UpdatedAt,
		&hs.Service.ID,
		&hs.Service.ServiceName,
		&hs.Service.Active,
		&hs.Service.Icon,
		&hs.Service.CreatedAt,
		&hs.Service.UpdatedAt,
		&hs.HostName,
	)

	if err != nil {
		log.Println(err)
		return hs, err
	}

	return hs, nil
}

func (m *postgresDBRepo) UpdateHostService(hs models.HostService) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE host_services
  	SET host_id = $1,
				service_id = $2,
				active = $3,
				schedule_number = $4,
				schedule_unit = $5,
				last_check = $6,
				last_message = $7,
				status = $8,
				updated_at = $9
		WHERE id = $10`

	_, err := m.DB.ExecContext(ctx, stmt, hs.HostID, hs.ServiceID, hs.Active, hs.ScheduleNumber, hs.ScheduleUnit, hs.LastCheck, hs.LastMessage, hs.Status, time.Now(), hs.ID)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (m *postgresDBRepo) GetServicesToMonitor() ([]models.HostService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.last_message, hs.status, hs.created_at, hs.updated_at, 
       s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at,
       h.host_name
		FROM host_services AS hs
		LEFT JOIN services AS s ON (hs.service_id = s.id)
		LEFT JOIN hosts AS h ON (hs.host_id = h.id)
		WHERE h.active = 1 AND hs.active = 1`

	rows, err := m.DB.QueryContext(ctx, stmt)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	var services []models.HostService

	for rows.Next() {
		var hs models.HostService
		err = rows.Scan(
			&hs.ID,
			&hs.HostID,
			&hs.ServiceID,
			&hs.Active,
			&hs.ScheduleNumber,
			&hs.ScheduleUnit,
			&hs.LastCheck,
			&hs.LastMessage,
			&hs.Status,
			&hs.CreatedAt,
			&hs.UpdatedAt,
			&hs.Service.ID,
			&hs.Service.ServiceName,
			&hs.Service.Active,
			&hs.Service.Icon,
			&hs.Service.CreatedAt,
			&hs.Service.UpdatedAt,
			&hs.HostName,
		)
		if err != nil {
			log.Println(err)
			return services, nil
		}
		services = append(services, hs)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return services, nil
	}

	return services, nil
}

func (m *postgresDBRepo) GetAllServiceStatusCounts() (models.ServiceStatusCount, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	SELECT 
		(SELECT count(id) FROM host_services WHERE active = 1 AND status = 'pending') as pending,
		(SELECT count(id) FROM host_services WHERE active = 1 AND status = 'healthy') as healthy,
		(SELECT count(id) FROM host_services WHERE active = 1 AND status = 'warning') as warning,
		(SELECT count(id) FROM host_services WHERE active = 1 AND status = 'problem') as problem`

	var counts models.ServiceStatusCount

	row := m.DB.QueryRowContext(ctx, query)
	err := row.Scan(
		&counts.Pending,
		&counts.Healthy,
		&counts.Warning,
		&counts.Problem,
	)
	if err != nil {
		return counts, err
	}

	return counts, nil
}
