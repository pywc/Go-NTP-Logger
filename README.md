# Go NTP Logger
 
Go NTP logger  functions as a logger to be used in conjunction with a preferred NTP server instance. 
You can use this for filtering NTP packets received from desired IP address ranges.

### Usage

1. Clone this repo
2. Configure the identifier and the network interface of the machine in `config/config.go`
3. Add a text file (e.g. `prefixes.txt`) consisting of IP prefixes desired to be logged
4. Run `./setup.sh`
5. `sudo su`
6. Run `go run .`
7. Detach from tmux by pressing `Ctrl-B / D`
8. See the pcap files in `output` (pcaps are rotated at midnight)
