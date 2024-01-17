package pgstorage

import (
	"context"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// execQuerierWrapper automatically adds a ctx timeout for the querier, also add before and after logs
type execQuerierWrapper struct {
	execQuerier
}

func (w *execQuerierWrapper) Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	logger := log.WithFields(utils.TraceID, ctx.Value(utils.CtxTraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[Exec], sql[%v], arguments[%v]", removeNewLine(sql), arguments)

	tag, err := w.execQuerier.Exec(ctx, sql, arguments...)

	logger.Debugf("DB query end, method[Exec], sql[%v] arguments[%v] rowsAffected[%v] err[%v] processTime[%v]",
		removeNewLine(sql), arguments, tag.RowsAffected(), err, time.Since(startTime).String())
	return tag, err
}

func (w *execQuerierWrapper) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	logger := log.WithFields(utils.TraceID, ctx.Value(utils.CtxTraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[Query], sql[%v], arguments[%v]", removeNewLine(sql), args)

	rows, err := w.execQuerier.Query(ctx, sql, args...)

	logger.Debugf("DB query end, method[Query], sql[%v] arguments[%v] rowCount[%v] err[%v] processTime[%v]", removeNewLine(sql), args, len(rows.RawValues()), err, time.Since(startTime).String())
	return rows, err
}

func (w *execQuerierWrapper) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	logger := log.WithFields(utils.TraceID, ctx.Value(utils.CtxTraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[QueryRow], sql[%v], arguments[%v]", removeNewLine(sql), args)

	row := w.execQuerier.QueryRow(ctx, sql, args...)

	logger.Debugf("DB query end, sql[%v] arguments[%v] method[QueryRow], processTime[%v]", removeNewLine(sql), args, time.Since(startTime).String())
	return row
}

func (w *execQuerierWrapper) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	logger := log.WithFields(utils.TraceID, ctx.Value(utils.CtxTraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[CopyFrom], tableName[%v]", tableName)

	res, err := w.execQuerier.CopyFrom(ctx, tableName, columnNames, rowSrc)

	logger.Debugf("DB query end, method[CopyFrom], tableName[%v] res[%v] err[%v] processTime[%v]", tableName, res, err, time.Since(startTime).String())
	return res, err
}

func removeNewLine(s string) string {
	return strings.Replace(s, "\n", " ", -1)
}
