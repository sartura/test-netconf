package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Juniper/go-netconf/netconf"
	"golang.org/x/crypto/ssh"
)

type config struct {
	Login    login
	UnitTest []unitTest
}

type login struct {
	Address  string
	Username string
	Password string
}

type unitTest struct {
	Name string
	Test []test
}

type testType int

const (
	stringMatch testType = 0
	regexMatch           = 1
)

func (t *testType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "stringMatch":
		*t = stringMatch
	case "regexMatch":
		*t = regexMatch
	default:
		*t = stringMatch
	}
	return nil
}

type test struct {
	Wait  int      //wait for n seconds
	RPC   string   //NETCONF command
	Reply string   //expected NETCONF reply in exact string or regex format
	Type  testType //reply match type
}

func parseConfig(configFile string, address string, username string, password string) {

	fmt.Printf("Parsing config file: %s\n", configFile)

	/* decode the toml config file */
	var Config config
	_, err := toml.DecodeFile(configFile, &Config)
	if err != nil {
		fmt.Printf("Toml error: %v\n", err)
		return
	}

	if "" == address {
		address = Config.Login.Address
	}
	if "" == username {
		username = Config.Login.Username
	}
	if "" == password {
		password = Config.Login.Password
	}

	auth := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	/* create NETCONF session */
	s, err := netconf.DialSSH(address, auth)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	/* iterate over the unit test's and execute them */
	for _, unitTest := range Config.UnitTest {
		fmt.Printf("run test %s\n", unitTest.Name)
		/* execute test's */
		for _, test := range unitTest.Test {
			time.Sleep(time.Duration(test.Wait) * time.Second)

			reply, err := s.Exec(netconf.RawMethod(test.RPC))
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
				fmt.Printf("Fail for test %s\n", test.RPC)
			}

			if reply != nil {
				switch test.Type {
				case regexMatch:
					r, err := regexp.Compile(test.Reply)
					if err != nil {
						fmt.Printf("Invalid regular expression, error: %s", err.Error())
						break
					}
					if !r.MatchString(reply.Data) {
						fmt.Printf("MISMATCH!\nEXPECTED: \n%s\nGOT: \n%s\n", test.Reply, reply.Data)
					}
				case stringMatch:
					if test.Reply != reply.Data {
						fmt.Printf("MISMATCH!\nEXPECTED: \n%s\nGOT: \n%s\n", test.Reply, reply.Data)
					}
				default:
					fmt.Println("Unknown test type: ", test.Type)
				}
			}
		}
	}
}

func main() {

	/* get list of config files */
	folder := flag.String("folder", ".", "Path to folder containing config files")
	address := flag.String("address", "", "IP address and port of the NETCONF server")
	username := flag.String("username", "", "NETCONF server usernamer")
	password := flag.String("password", "", "NETCONF server password")
	flag.Parse()

	parseConfig(*folder, *address, *username, *password)
}
