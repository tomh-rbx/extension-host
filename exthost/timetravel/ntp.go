package timetravel

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
)

func AdjustNtpTrafficRules(allowNtpTraffic bool) error {
	if allowNtpTraffic {
		err := common.ExecuteIpTablesCommand("-A", "OUTPUT", "-p", "udp", "--dport", "123", "-j", "ACCEPT")
		if err != nil {
			log.Error().Err(err).Msg("Failed to execute iptables command")
			return err
		}
		err = common.ExecuteIpTablesCommand("-A", "OUTPUT", "-p", "udp", "--sport", "123", "-j", "ACCEPT")
		if err != nil {
			log.Error().Err(err).Msg("Failed to execute iptables command")
			return err
		}
	} else {
		err := common.ExecuteIpTablesCommand("-A", "OUTPUT", "-p", "udp", "--dport", "123", "-j", "DROP")
		if err != nil {
			log.Error().Err(err).Msg("Failed to execute iptables command")
			return err
		}
		err = common.ExecuteIpTablesCommand("-A", "OUTPUT", "-p", "udp", "--sport", "123", "-j", "DROP")
		if err != nil {
			log.Error().Err(err).Msg("Failed to execute iptables command")
			return err
		}
	}
	return nil
}
