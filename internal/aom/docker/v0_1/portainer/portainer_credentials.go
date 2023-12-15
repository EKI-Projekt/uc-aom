package portainer

import (
	"bufio"
	"log"
	"os"
	"strings"
	"u-control/uc-aom/internal/pkg/utils"
)

type PortainerUserCredentials struct {
	Username string
	Password string
}

type appConfigProperties map[string]string

func GetPortainerCredentials() (*PortainerUserCredentials, error) {
	creds, err := readPortainerCredentials()
	if err != nil {
		return nil, err
	}
	return creds, nil
}

func createPropertiesMap(scanner bufio.Scanner) (appConfigProperties, error) {
	config := appConfigProperties{}
	for scanner.Scan() {
		line := scanner.Text()
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				config[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return nil, err
	}
	return config, nil
}

func readPortainerCredentials() (*PortainerUserCredentials, error) {

	portainerFilepath := utils.GetEnv("PORTAINER_CE_ENV_FILEPATH", "/var/lib/portainer-ce/portainer.env")
	file, err := os.Open(portainerFilepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := *bufio.NewScanner(file)

	props, err := createPropertiesMap(scanner)
	if err != nil {
		return nil, err
	}

	creds := PortainerUserCredentials{
		Username: props["PORTAINER_LOCAL_ADMIN_USER"],
		Password: props["PORTAINER_LOCAL_ADMIN_PW"],
	}

	return &creds, nil
}
