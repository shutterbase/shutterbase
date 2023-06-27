package tracing

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/shutterbase/shutterbase")

func GetTracer() trace.Tracer {
	return tracer
}

func ZerologTraceHook(ctx context.Context) zerolog.HookFunc {
	return func(e *zerolog.Event, level zerolog.Level, message string) {
		if level == zerolog.NoLevel {
			return
		}
		if !e.Enabled() {
			return
		}

		if ctx == nil {
			return
		}

		span := trace.SpanFromContext(ctx)
		if !span.IsRecording() {
			return
		}

		// code from: https://github.com/uptrace/opentelemetry-go-extra/tree/main/otellogrus
		// whose license(BSD 2-Clause) can be found at: https://github.com/uptrace/opentelemetry-go-extra/blob/v0.1.18/LICENSE

		// Unlike logrus or exp/slog, zerolog does not give hooks the ability to get the whole event/message with all its key-values
		// see: https://github.com/rs/zerolog/issues/300

		attrs := make([]attribute.KeyValue, 0)

		logSeverityKey := attribute.Key("log.severity")
		logMessageKey := attribute.Key("log.message")
		attrs = append(attrs, logSeverityKey.String(level.String()))
		attrs = append(attrs, logMessageKey.String(message))

		// todo: add caller info.

		span.AddEvent("log", trace.WithAttributes(attrs...))
		if level >= zerolog.ErrorLevel {
			span.SetStatus(codes.Error, message)
		}

	}
}
