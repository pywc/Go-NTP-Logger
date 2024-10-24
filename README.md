# Go NTP Logger
 
This tool acts as a NTP server and a logger. You can use this tool for logging the NTP packets received from your desired IP prefixes.

### Usage

1. Install `go`
2. Configure the IP address and the identifier of the machine in `config/config.go`
3. Create a text file (e.g. `prefixes.txt`) consisting of IP prefixes desired to be logged
4. Run `./setup.sh`
5. `sudo su`
6. Run `go run .`
7. Detach from tmux by pressing `Ctrl-B / D`
8. Fun and profit

### Example Configuration

```
package config

// Server config
var SERVER_NAME = "UCSD Sysnet"
var SERVER_VERSION = "v1.0"
var SERVER_IP = "0.0.0.0"
var SERVER_PORT = 123

// IO config
var IP_PREFIX_FILE = "prefixes.txt"
var OUTPUT_FILE_PREFIX = "packets/" + "ucsd_sysnet" // do not add ./ at front

// NTP config
var NTP_REF_ID = []byte{132, 239, 1, 6} // UCSD GPS IP
var NTP_STRATUM = 2                     // 2: secondary server
var NTP_POLL_INTERVAL = 4               // default
var NTP_PRECISION = 0xF6                // -10
```
