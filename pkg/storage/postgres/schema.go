package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
)

const Schema = `
CREATE SCHEMA IF NOT EXISTS gulfstream;

CREATE TABLE IF NOT EXISTS gulfstream.versions
(
    stream_id      uuid         NOT NULL,
    stream_name    VARCHAR(128) NOT NULL,
    version integer,
    PRIMARY KEY (stream_name, stream_id)
);

CREATE TABLE IF NOT EXISTS gulfstream.states
(
    stream_id      uuid         NOT NULL,
    stream_name    VARCHAR(128) NOT NULL,
    version integer,
    raw_data  bytea,
    PRIMARY KEY (stream_name, stream_id)
);

CREATE TABLE IF NOT EXISTS gulfstream.events
(
    stream_id        uuid         NOT NULL,
    stream_name      VARCHAR(128) NOT NULL,
    event_name      VARCHAR(256) NOT NULL,
    version    integer,
    created_at BIGINT,
    raw_data    bytea,
    PRIMARY KEY (stream_name, stream_id, version)
);

CREATE TABLE IF NOT EXISTS gulfstream.outbox
(
    stream_id        uuid         NOT NULL,
    stream_name      VARCHAR(128) NOT NULL,
    version integer,
    raw_data    bytea,
    PRIMARY KEY (stream_name, stream_id, version)
);
`

func CreateSchema(ctx context.Context, pool *pgxpool.Pool) error {
	queries := strings.Split(Schema, ";")
	for _, query := range queries {
		if len(query) == 0 {
			continue
		}

		query = strings.TrimSpace(query)
		query = strings.ReplaceAll(query, "\n", "")
		query = query + ";"

		_, err := pool.Exec(ctx, query)
		if err != nil {
			return err
		}
	}
	return nil
}

func DropSchema(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, selectTablesSQL)
	if err != nil {
		return err
	}
	defer rows.Close()
	tables := make([]string, 0, 12)
	var tableName string
	for rows.Next() {
		if err := rows.Scan(&tableName); err != nil {
			return err
		}
		tables = append(tables, "gulfstream."+tableName)
	}
	sql := dropTableSQL + " " + strings.Join(tables, ",")
	_, err = pool.Exec(ctx, sql)
	return err
}

const (
	selectTablesSQL = `
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'gulfstream'`

	dropTableSQL = `DROP TABLE IF EXISTS`

	insertVersionSQL = `
INSERT INTO gulfstream.versions (stream_name, stream_id, version) 
VALUES ($1, $2, $3)`

	updateVersionSQL = `
UPDATE gulfstream.versions 
SET version=$1 
WHERE stream_name=$2 AND stream_id=$3 AND version=$4`

	insertEventSQL = `
INSERT INTO gulfstream.events (stream_id, stream_name, event_name, version, created_at, raw_data) 
VALUES ($1, $2, $3, $4, $5, $6)`

	insertStateSQL = `
INSERT INTO  gulfstream.states (stream_name, stream_id, version, raw_data) 
VALUES ($1, $2, $3, $4)`

	updateStateSQL = `
UPDATE gulfstream.states SET version=$1, raw_data=$2 
WHERE stream_name=$3 AND stream_id=$4`

	selectStateSQL = `
SELECT raw_data
FROM gulfstream.states
WHERE stream_name=$1 AND stream_id=$2`

	insertOutboxSQL = `
INSERT INTO gulfstream.outbox (stream_id, stream_name, version, raw_data) 
VALUES ($1, $2, $3, $4)`

	deleteOutboxSQL = `DELETE FROM gulfstream.outbox WHERE stream_name=$1 AND stream_id=$2 AND version <= $3`
)
