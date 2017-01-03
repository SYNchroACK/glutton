package protocols

import (
	"honnef.co/go/netdb"
)

// GetHandler returns destination address of the service to redirect traffic
//func GetHandler(p int) string {
//	return portConf.Ports[p]
//}

// GetDefaultHandler returns the default handler or empty string
//func GetDefaultHandler() string {
//	return portConf.Default
//}

// GetProtocol (80, "tcp")
func GetProtocol(port int, transport string) *netdb.Servent {
	prot := netdb.GetProtoByName(transport)
	return netdb.GetServByPort(port, prot)
}
