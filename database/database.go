package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

func LogPgxPoolStatistics(ctx context.Context, msg string) {
	db := ctx.Value("db").(*pgxpool.Pool)
	stat := db.Stat()
	fmt.Printf("pgxstat (%s):\n{\n\tAcquireCount: %d,\n\tAcquireDuration: %d,\n\tAcquiredConns: %d,\n\tCanceledAcquireCount: %d,\n\tConstructingConns: %d,\n\tEmptyAcquireCount: %d,\n\tIdleConns: %d,\n\tMaxConns: %d,\n\tTotalConns:%d,\n\tNewConnsCount:%d,\n\tMaxIdleDestroyCount:%d,\n\tMaxLifetimeDestroyCount:%d\n}\n", msg, stat.AcquireCount(), stat.AcquireDuration(), stat.AcquiredConns(), stat.CanceledAcquireCount(), stat.ConstructingConns(), stat.EmptyAcquireCount(), stat.IdleConns(), stat.MaxConns(), stat.TotalConns(), stat.NewConnsCount(), stat.MaxIdleDestroyCount(), stat.MaxLifetimeDestroyCount())
}
