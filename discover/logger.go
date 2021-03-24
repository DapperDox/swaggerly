package discover

import (
	"github.com/sirupsen/logrus"

	"github.com/kenjones-cisco/dapperdox/logger"
)

func log() logrus.FieldLogger {
	return logger.Logger().WithField("pkg", "discover")
}
