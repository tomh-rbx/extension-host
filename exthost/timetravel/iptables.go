package timetravel

import (
  "github.com/rs/zerolog/log"
  "os/exec"
  "strings"
)

func useIptablesLegacy() bool {
  //execute command
  out, err := exec.Command("iptables", "-V").Output()
  if err != nil {
    log.Error().Err(err).Msg("Failed to execute iptables -V")
    return false
  }
  //check if output contains "nf_tables"
  return strings.Contains(string(out), "nf_tables")
}

func executeIpTablesCommand(useIptablesLegacy bool, iptablesCmd string, args ...string) error {
  cmd := exec.Command(iptablesCmd, args...)
  if useIptablesLegacy {
    cmd.Env = append(cmd.Env, "XTABLES_LOCKFILE=/tmp/xtables.lock")
  }
  out, err := cmd.CombinedOutput()
  if err != nil {
    log.Error().Err(err).Str("output", string(out)).Msg("Failed to execute iptables command")
    return err
  }
  return nil
}
func adjustNtpTrafficRules(allowNtpTraffic bool) error {
  useIptablesLegacy := useIptablesLegacy()
  iptablesCmd := "iptables"
  if useIptablesLegacy {
    iptablesCmd = "iptables-legacy"
  }
  if allowNtpTraffic {
    err := executeIpTablesCommand(useIptablesLegacy, iptablesCmd, "-A", "OUTPUT", "-p", "udp", "--dport", "123", "-j", "ACCEPT")
    if err != nil {
      log.Error().Err(err).Msg("Failed to execute iptables command")
      return err
    }
    err = executeIpTablesCommand(useIptablesLegacy, iptablesCmd, "-A", "OUTPUT", "-p", "udp", "--sport", "123", "-j", "ACCEPT")
    if err != nil {
      log.Error().Err(err).Msg("Failed to execute iptables command")
      return err
    }
  } else {
    err := executeIpTablesCommand(useIptablesLegacy, iptablesCmd, "-D", "OUTPUT", "-p", "udp", "--dport", "123", "-j", "DROP")
    if err != nil {
      log.Error().Err(err).Msg("Failed to execute iptables command")
      return err
    }
    err = executeIpTablesCommand(useIptablesLegacy, iptablesCmd, "-D", "OUTPUT", "-p", "udp", "--sport", "123", "-j", "DROP")
    if err != nil {
      log.Error().Err(err).Msg("Failed to execute iptables command")
      return err
    }
  }
  return nil
}
