package constants

const (
	DefaultHubPort      = 4242
	DefaultAgentPort    = 8000
	DefaultServiceName  = "gp"
	ConfigFileName      = "gp.conf"
	ShellPath           = "/bin/bash"
	GpSSH               = "gpssh"
	MaxRetries          = 10
	PlatformDarwin      = "darwin"
	PlatformLinux       = "linux"
	DefaultQdMaxConnect = 150
	QeConnectFactor     = 3
	DefaultBuffer       = "128000kB"
	OsOpenFiles         = 65535
	DefaultDatabase     = "template1"
	DefaultEncoding     = "UTF-8"
	EtcHostsFilepath    = "/etc/hosts"
	ReplicationSlotName = "internal_wal_replication_slot"
	DefaultStartTimeout = 600
	DefaultPostgresLogDir = "log"
	GroupMirroring      = "group"
	SpreadMirroring     = "spread"
	DefaultSegName      = "gpseg"
)

// gp_segment_configuration specific constants
const (
	RolePrimary = "p"
	RoleMirror   = "m"
)

// Catalog tables
const (
	GpSegmentConfiguration = "gp_segment_configuration"
)
