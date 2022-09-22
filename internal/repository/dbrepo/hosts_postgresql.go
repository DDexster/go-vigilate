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

		serviceQuery := `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.status, hs.created_at, hs.updated_at, 
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
	stmt = `INSERT INTO host_services (host_id, service_id, active, schedule_number, schedule_unit, status, created_at, updated_at) 
		VALUES($1, 1, 0, 3, 'm', 'pending', $2, $3)`

	_, err = m.DB.ExecContext(ctx, stmt,
		newId,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		log.Println(err)
		return newId, err
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
	stmt = `SELECT hs.id, hs.host_id, hs.service_id, hs.active, hs.schedule_number, hs.schedule_unit, hs.last_check, hs.status, hs.created_at, hs.updated_at, 
       s.id, s.service_name, s.active, s.icon, s.created_at, s.updated_at
		FROM host_services AS hs
		LEFT JOIN services AS s ON (hs.service_id = s.id)
		WHERE hs.host_id = $1`

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
