package route

import (
	"database/sql"
	"log"

	"google.golang.org/grpc"

	tasksDomain "lms-task-service/internal/domain/tasks"
	"lms-task-service/internal/pkg/db/redis"
	tasksPb "lms-task-service/pb/tasks"
)

// GrpcRoute func
func GrpcRoute(grpcServer *grpc.Server, db *sql.DB, log *log.Logger, cache *redis.Cache) {
	taskServer := tasksDomain.TaskServiceServer{Db: db, Cache: cache, Log: log}
	tasksPb.RegisterTaskServiceServer(grpcServer, &taskServer)
}
