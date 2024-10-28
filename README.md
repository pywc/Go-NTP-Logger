# Go NTP Logger
 
This tool is intended to be used as a packet logger to be used in conjunction with a preferred NTP server instance. 
You can use this for filtering NTP packets received from desired IP address ranges.

### Setup

1. Clone this repo
2. Install and configure dependencies
    - Option 1: run `setup.sh` which installs `golang`, `libpcap-dev`. `chrony`, and `tmux`, activates the `chronyd` service, and sets up a `tmux` session
    - Option 2: install `golang`, `libpcap-dev`, an NTP service, and a terminal multiplexer of your choice
3. Specify the identifier and the network interface of the machine in `config/config.go`
4. Add a text file (e.g. `prefixes.txt`) consisting of IP prefixes desired to be logged
5. Run `sudo su`
6. Run `go run .`
7. Detach from the terminal multiplexer (e.g. `tmux`)
8. See the pcap files in the `output` folder (pcap files are rotated every midnight)
