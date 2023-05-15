package timetravel

import (
  "errors"
  "github.com/rs/zerolog/log"
	"syscall"
	"time"
)

func AdjustTime(offset time.Duration, negate bool) error {
	tp := syscall.Timeval{}
	err := syscall.Gettimeofday(&tp)
  initialTime := tp.Sec
	log.Info().Msgf("Current time: %d", tp.Sec)
	if err != nil {
		log.Err(err).Msg("Could not change time offset - Gettimeofday")
		return err
	}
	seconds := int64(offset) / int64(time.Second)
	if negate {
		seconds = -seconds
	}
	log.Info().Msgf("Adjusting time by %d seconds", seconds)
	tp.Sec += seconds
	err = syscall.Settimeofday(&tp)
	if err != nil {
		log.Err(err).Msg("Could not change time offset - Settimeofday")
		return err
	}
  newTime := tp.Sec
	log.Info().Msgf("New time: %d", tp.Sec)
  diff := newTime - initialTime
  if float64(diff) >= (offset.Seconds() -2)  {
    return errors.New("time offset not applied")
  }
	return nil
}
