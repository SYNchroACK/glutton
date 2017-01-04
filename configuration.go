package glutton

import (
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "gopkg.in/yaml.v2"
)

// Config For the fields of ports.conf

type Config struct {
    Default string
    Ports   map[int]string
}

var (
    portConf Config

    src     []string // slice contains attributes of previous packet conntrack logs
    desP    int      // Destination port of previous packet returned to the UDP server
    unknown []string // Address not logged by conntrack
)

// LoadPorts ports.yml file into portConf
func LoadPorts(confPath string) {
    f, err := filepath.Abs(confPath)
    if err != nil {
        log.Println("Error in absolute representation of file LoadPorts().")
        os.Exit(1)
    }
    ymlF, err := ioutil.ReadFile(f)

    if err != nil {
        panic(err)
    }

    err = yaml.Unmarshal(ymlF, &portConf)
    if err != nil {
        CheckError("[*] service.yml unmarshal Error.", err)
    }

    if len(portConf.Ports) == 0 {
        log.Println("Host list is empty, Please update ports.yml")
        os.Exit(1)
    }
    log.Println("Port configuration loaded successfully....")

}
