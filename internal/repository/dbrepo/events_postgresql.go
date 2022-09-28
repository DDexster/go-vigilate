package dbrepo

import (
	"context"
	"github.com/DDexster/go-vigilate/internal/models"
	"log"
	"time"
)

func (m *postgresDBRepo) InsertEvent(e models.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO events 
    (event_type, host_service_id, host_id, service_name, host_name, message, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := m.DB.ExecContext(ctx, stmt,
		e.EventType,
		e.HostServiceID,
		e.HostID,
		e.ServiceName,
		e.HostName,
		e.Message,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (m *postgresDBRepo) GetAllEvents() ([]models.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT id, event_type, host_service_id, host_id, service_name, host_name, message, created_at, updated_at
		FROM events
		ORDER BY created_at DESC`

	rows, err := m.DB.QueryContext(ctx, stmt)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	var events []models.Event

	for rows.Next() {
		var e models.Event

		err := rows.Scan(
			&e.ID,
			&e.EventType,
			&e.HostServiceID,
			&e.HostID,
			&e.ServiceName,
			&e.HostName,
			&e.Message,
			&e.CreatedAt,
			&e.UpdatedAt,
		)

		if err != nil {
			log.Println(err)
			return events, err
		}

		events = append(events, e)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return events, err
	}

	return events, nil
}
