package logwriter

import (
	"LoggingService/config"
)

type LogWriterSettings struct {
	Config      config.LogfileSettings
	ColumnOrder []string
}
