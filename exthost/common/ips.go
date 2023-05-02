package common

import (
	"github.com/rs/zerolog/log"
	"net"
)

func GetIP4sAndNICs() ([]string, []string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get interfaces")
		return []string{}, []string{}
	}
	resultIP4s := make([]string, 0)
	resultNICs := make([]string, 0)
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		resultNICs = append(resultNICs, i.Name)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get addresses")
			break
		}
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// process IP address
			if !ip.IsLoopback() && ip.To4() != nil {
				resultIP4s = append(resultIP4s, ip.String())
			}
		}
	}
	return resultIP4s, resultNICs
}
