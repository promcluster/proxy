package log

import (
	"os"
	"runtime"
	"time"

	"github.com/mattn/go-isatty"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var defaultLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)

// InitLogger init a logger
func InitLogger(cfg Config) (*zap.Logger, error) {
	w := os.Stdout
	encConf := zap.NewProductionEncoderConfig()
	encConf.EncodeTime = SimpleTimeEncoder

	// parse logging level
	if err := defaultLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, err
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encConf),
		w,
		defaultLevel,
	)
	logger := zap.New(core)
	return logger, nil
}

// SimpleTimeEncoder serializes a time.Time to a simplified format without timezone.
func SimpleTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// IsTerminal checks if the stdOut is a terminal or not.
func IsTerminal(f *os.File) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
