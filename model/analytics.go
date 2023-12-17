package model

import (
	"context"
	glContext "dev.hackerman.me/artheon/veverse-shared/context"
	"encoding/json"
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gofrs/uuid"
	googleUUID "github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type AnalyticEvent struct {
	Id                uuid.UUID `json:"id,omitempty"`
	AppId             uuid.UUID `json:"appId,omitempty"`
	ContextEntityId   uuid.UUID `json:"contextEntityId,omitempty"`
	ContextEntityType string    `json:"contextEntityType,omitempty"`
	UserId            uuid.UUID `json:"userId,omitempty"`
	Platform          string    `json:"platform,omitempty"`
	Deployment        string    `json:"deployment,omitempty"`
	Configuration     string    `json:"configuration,omitempty"`
	Event             string    `json:"event,omitempty"`
	Timestamp         time.Time `json:"timestamp,omitempty"`
	Payload           string    `json:"data,omitempty"`
}

func (e AnalyticEvent) String() string {
	var out = "{"
	out += "id: " + e.Id.String() + ", "
	out += "appId: " + e.AppId.String() + ", "
	out += "contextEntityId: " + e.ContextEntityId.String() + ", "
	out += "contextEntityType: " + e.ContextEntityType + ", "
	out += "userId: " + e.UserId.String() + ", "
	out += "platform: " + e.Platform + ", "
	out += "deployment: " + e.Deployment + ", "
	out += "configuration: " + e.Configuration + ", "
	out += "event: " + e.Event + ", "
	out += "timestamp: " + e.Timestamp.Format("2006-01-02 15:04:05") + ", "
	out += "payload: " + e.Payload + ", "
	out += "}"
	return out
}

type AnalyticEventBatch Batch[AnalyticEvent]

type IndexAnalyticEventRequest struct {
	Offset            *int64  `json:"offset,omitempty"`
	Limit             *int64  `json:"limit,omitempty"`
	AppId             *string `json:"appId,omitempty"`
	ContextEntityId   *string `json:"contextId,omitempty"`
	ContextEntityType *string `json:"contextType,omitempty"`
	UserId            *string `json:"userId,omitempty"`
	Platform          *string `json:"platform,omitempty"`
	Deployment        *string `json:"deployment,omitempty"`
	Configuration     *string `json:"configuration,omitempty"`
	Event             *string `json:"event,omitempty"`
}

func IndexAnalyticEvent(ctx context.Context, requester *User, request IndexAnalyticEventRequest) (b *AnalyticEventBatch, err error) {
	if requester == nil || requester.Id == uuid.Nil {
		return nil, ErrNoRequester
	}

	c, ok := ctx.Value(glContext.Clickhouse).(driver.Conn)
	if !ok || c == nil {
		return nil, ErrNoDatabase
	}

	var batch = AnalyticEventBatch{
		Offset: 0,
		Limit:  100,
		Total:  0,
	}

	if request.Offset != nil && *request.Offset >= 0 {
		batch.Offset = *request.Offset
	}

	if request.Limit != nil && *request.Limit > 0 && *request.Limit <= 100 {
		batch.Limit = *request.Limit
	}

	var (
		qt       string
		qtArgNum int
		q        string
		qArgNum  int
		rows     driver.Rows
	)

	// fixme: NOT CLICKHOUSE NEED TO CHECK IF USER HAS ACCESS TO APP
	//	if !requester.IsAdmin {
	//		// if the user is not an admin, they can only see analytics for applications they own
	//		if request.AppId == nil {
	//			// by Vasily's request
	//			return nil, ErrNoPermission
	//		}
	//
	//		qa := `select is_owner
	//from accessibles ac
	//where ac.entity_id = $1
	//  and ac.user_id = $2
	//  and ac.is_owner = true `
	//		// fixme: NOT CLICKHOUSE, WRONG
	//		rows, err = c.Query(ctx, qa, request.AppId, requester.Id)
	//		if err != nil {
	//			logrus.Error(err)
	//			// by Vasily's request
	//			return nil, ErrNoPermission
	//		}
	//
	//		defer func(rows driver.Rows) {
	//			err := rows.Close()
	//			if err != nil {
	//				logrus.Error(err)
	//			}
	//		}(rows)
	//
	//		var (
	//			isOwner bool
	//		)
	//
	//		for rows.Next() {
	//			err = rows.Scan(&isOwner)
	//			if err != nil {
	//				logrus.Error(err)
	//				return nil, ErrNoPermission
	//			}
	//
	//			if isOwner {
	//				break
	//			}
	//		}
	//
	//		if !isOwner {
	//			return nil, ErrNoPermission
	//		}
	//	}

	qt = `select count(*) from events where true`
	q = `select id,
       appId,
       contextEntityId,
       contextEntityType,
       userId,
       platform,
       deployment,
       configuration,
       event,
       timestamp,
       payload
from events
where true`

	args := make([]interface{}, 0)

	var appIdUuid uuid.UUID
	if request.AppId != nil {
		appIdUuid = uuid.FromStringOrNil(*request.AppId)
	}
	if request.AppId != nil && !appIdUuid.IsNil() {
		qtArgNum++
		qt += ` and appId = $` + strconv.Itoa(qtArgNum)
		qArgNum++
		q += ` and appId = $` + strconv.Itoa(qArgNum)
		args = append(args, *request.AppId)
	}

	var contextIdUuid uuid.UUID
	if request.AppId != nil {
		if request.ContextEntityId != nil {
			contextIdUuid = uuid.FromStringOrNil(*request.ContextEntityId)
		} else {
			contextIdUuid = uuid.Nil
		}
	}
	if request.ContextEntityId != nil && !contextIdUuid.IsNil() {
		qtArgNum++
		qt += ` and contextEntityId = $` + strconv.Itoa(qtArgNum)
		qArgNum++
		q += ` and contextEntityId = $` + strconv.Itoa(qArgNum)
		args = append(args, *request.ContextEntityId)
	}
	if request.ContextEntityType != nil && *request.ContextEntityType != "" {
		qtArgNum++
		qt += ` and contextEntityType = $` + strconv.Itoa(qtArgNum)
		qArgNum++
		q += ` and contextEntityType = $` + strconv.Itoa(qArgNum)
		args = append(args, *request.ContextEntityType)
	}

	var userIdUuid uuid.UUID
	if request.AppId != nil {
		userIdUuid = uuid.FromStringOrNil(*request.UserId)
	}
	if request.UserId != nil && !userIdUuid.IsNil() {
		qtArgNum++
		qt += ` and userId = $` + strconv.Itoa(qtArgNum)
		qArgNum++
		q += ` and userId = $` + strconv.Itoa(qArgNum)
		args = append(args, *request.UserId)
	}
	if request.Platform != nil && *request.Platform != "" {
		qtArgNum++
		qt += ` and platform = $` + strconv.Itoa(qtArgNum)
		qArgNum++
		q += ` and platform = $` + strconv.Itoa(qArgNum)
		args = append(args, *request.Platform)
	}
	if request.Deployment != nil && *request.Deployment != "" {
		qtArgNum++
		qt += ` and deployment = $` + strconv.Itoa(qtArgNum)
		qArgNum++
		q += ` and deployment = $` + strconv.Itoa(qArgNum)
		args = append(args, *request.Deployment)
	}
	if request.Configuration != nil && *request.Configuration != "" {
		qtArgNum++
		qt += ` and configuration = $` + strconv.Itoa(qtArgNum)
		qArgNum++
		q += ` and configuration = $` + strconv.Itoa(qArgNum)
		args = append(args, *request.Configuration)
	}
	if request.Event != nil && *request.Event != "" {
		qtArgNum++
		qt += " and event = $" + strconv.Itoa(qtArgNum)
		qArgNum++
		q += " and event = $" + strconv.Itoa(qArgNum)
		eventArg := *request.Event
		args = append(args, eventArg)
	}

	args = append(args, batch.Limit)
	args = append(args, batch.Offset)
	q += ` order by timestamp desc limit $` + strconv.Itoa(qArgNum+1) + ` offset $` + strconv.Itoa(qArgNum+2)

	err = c.QueryRow(ctx, qt, args...).Scan(&batch.Total)
	if err != nil {
		return nil, err
	}

	if batch.Total == 0 {
		return &batch, nil
	}

	rows, err = c.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}

	defer func(rows driver.Rows) {
		err := rows.Close()
		if err != nil {
			logrus.Errorf("failed to close rows: %v", err)
		}
	}(rows)

	for rows.Next() {
		var event AnalyticEvent
		err = rows.Scan(&event.Id, &event.AppId, &event.ContextEntityId, &event.ContextEntityType, &event.UserId, &event.Platform, &event.Deployment, &event.Configuration, &event.Event, &event.Timestamp, &event.Payload)
		if err != nil {
			return nil, err
		}
		batch.Entities = append(batch.Entities, event)
	}

	return &batch, nil
}

type AnalyticEventRequest struct {
	AppId             uuid.UUID `json:"appId" query:"app-id"`
	ContextEntityId   uuid.UUID `json:"contextEntityId" query:"context-entity-id"`
	ContextEntityType string    `json:"contextEntityType" query:"context-entity-type"`
	UserId            uuid.UUID `json:"userId" query:"user-id" validate:"required"`
	Platform          string    `json:"platform" query:"platform" validate:"required"`
	Deployment        string    `json:"deployment" query:"deployment" validate:"required"`
	Configuration     string    `json:"configuration" query:"configuration" validate:"required"`
	Event             string    `json:"event" query:"event" validate:"required"`
	Payload           any       `json:"data" query:"data" validate:"required"`
}

func convertUuid(u uuid.UUID) googleUUID.UUID {
	return googleUUID.UUID{
		u[0],
		u[1],
		u[2],
		u[3],
		u[4],
		u[5],
		u[6],
		u[7],
		u[8],
		u[9],
		u[10],
		u[11],
		u[12],
		u[13],
		u[14],
		u[15],
	}
}

func ReportEvent(ctx context.Context, requester *User, event AnalyticEventRequest) error {
	if requester == nil {
		return fmt.Errorf("requester is nil")
	}

	clickhouse, ok := ctx.Value(glContext.Clickhouse).(driver.Conn)
	if !ok {
		return fmt.Errorf("failed to get clickhouse client from context")
	}

	var userId googleUUID.UUID
	if requester.IsInternal {
		userId = convertUuid(event.UserId)
	} else {
		userId = convertUuid(requester.Id)
	}

	bytes, err := json.Marshal(event.Payload)
	if err != nil {
		logrus.Errorf("failed to marshal payload: %v", err)
		return fmt.Errorf("failed to report event")
	}

	q := `INSERT INTO events (appId, contextEntityId, contextEntityType, userId, platform, deployment, configuration, event, payload)
	VALUES	($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	err = clickhouse.Exec(ctx, q,
		event.AppId,
		event.ContextEntityId,
		event.ContextEntityType,
		userId,
		event.Platform,
		event.Deployment,
		event.Configuration,
		event.Event,
		string(bytes),
	)

	if err != nil {
		logrus.Errorf("failed to insert: %v", err)
		return fmt.Errorf("failed to report event")
	}

	return nil
}

type SystemLogRequest struct {
	Service   string    `json:"service" query:"service"`
	Timestamp time.Time `json:"timestamp" query:"timestamp"`
	Level     string    `json:"level" query:"level"`
	Message   string    `json:"message" query:"message"`
	Payload   any       `json:"data" query:"data"`
}

func ReportSystemLog(ctx context.Context, event SystemLogRequest) error {
	clickhouse, ok := ctx.Value(glContext.Clickhouse).(driver.Conn)
	if !ok {
		return fmt.Errorf("failed to get clickhouse client from context")
	}

	bytes, err := json.Marshal(event.Payload)
	if err != nil {
		logrus.Errorf("failed to marshal payload: %v", err)
		return fmt.Errorf("failed to report event")
	}

	q := `INSERT INTO syslog (service, message, level, payload)
	VALUES	($1, $2, $3, $4)`
	err = clickhouse.Exec(ctx, q,
		event.Service,
		event.Message,
		event.Level,
		string(bytes),
	)

	if err != nil {
		logrus.Errorf("failed to insert: %v", err)
		return fmt.Errorf("failed to report event")
	}

	return nil
}
