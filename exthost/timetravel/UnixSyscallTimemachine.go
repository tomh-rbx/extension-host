package timetravel

import (
  "github.com/rs/zerolog/log"
  "github.com/steadybit/extension-host/exthost"
  "syscall"
  "time"
)

func AdjustTime(offset time.Duration, negate bool) error {
  tp := syscall.Timeval{}
  err := syscall.Gettimeofday(&tp)
  if err != nil {
    log.Err(err).Msg("Could not change time offset - Gettimeofday")
    return err
  }
  seconds := exthost.ToInt64(offset.Seconds)
  if negate {
    seconds = -seconds
  }
  tp.Sec += seconds
  err = syscall.Settimeofday(&tp)
  if err != nil {
    log.Err(err).Msg("Could not change time offset - Settimeofday")
    return err
  }
  return nil
}
