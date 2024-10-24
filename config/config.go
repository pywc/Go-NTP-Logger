package config

// Server config
var SERVER_NAME = "UCSD Sysnet"
var SERVER_VERSION = "v1.0"
var SERVER_IP = "137.110.222.27"
var SERVER_PORT = 123

// IO config
var IP_PREFIX_FILE = "prefixes.txt"
var OUTPUT_FILE_PREFIX = "ucsd_sysnet" // do not add ./ at front

// NTP config
var NTP_REF_ID = []byte{132, 239, 1, 6} // UCSD GPS IP
var NTP_STRATUM = 2                     // 2: secondary server
var NTP_POLL_INTERVAL = 4               // default
var NTP_PRECISION = 0xF6                // -10
