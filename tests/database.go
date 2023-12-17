package tests

import (
	"context"
	glContext "dev.hackerman.me/artheon/veverse-shared/context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strconv"
)

func GetDatabaseContext(ctx context.Context) (context.Context, error) {
	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	user := os.Getenv("DATABASE_USER")
	pass := os.Getenv("DATABASE_PASS")
	name := os.Getenv("DATABASE_NAME")
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, name)

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	ctx = context.WithValue(ctx, glContext.Database, pool)

	return ctx, nil
}

func GetClickhouseContext(ctx context.Context) (context.Context, error) {
	clickhouseHost := os.Getenv("CLICKHOUSE_HOST")
	clickhousePort := os.Getenv("CLICKHOUSE_PORT")
	clickhouseUser := os.Getenv("CLICKHOUSE_USER")
	clickhousePass := os.Getenv("CLICKHOUSE_PASS")
	clickhouseName := os.Getenv("CLICKHOUSE_NAME")
	clickhousePortNum, err := strconv.Atoi(clickhousePort)
	if err != nil {
		return ctx, fmt.Errorf("failed to convert clickhouse port to int: %w", err)
	}
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?debug=true", clickhouseUser, clickhousePass, clickhouseHost, clickhousePortNum, clickhouseName)
	options, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return ctx, fmt.Errorf("failed to parse clickhouse dsn: %w", err)
	}

	conn, err := clickhouse.Open(options)
	if err != nil || conn == nil {
		return ctx, fmt.Errorf("failed to connect to clickhouse")
	}

	err = conn.Ping(ctx)
	if err != nil {
		return ctx, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	if conn == nil {
		return ctx, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	ctx = context.WithValue(ctx, glContext.Clickhouse, conn)

	return ctx, nil
}
