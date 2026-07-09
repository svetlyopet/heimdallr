package constants

const (
	AppDefaultName = "heimdallr"

	ApiDefaultPort = "8080"
	ApiDefaultHost = "localhost"
)

var (
	AppDefaultTrustedProxies = []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
)
