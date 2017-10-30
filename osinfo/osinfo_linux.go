package osinfo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	entRelVerRgx = regexp.MustCompile(`/d+(\./d+)?(\./d+)?`)
)

const (
	osRelease = "/etc/os-release"
	oRelease  = "/etc/oracle-release"
	rhRelease = "/etc/redhat-release"
)

func exists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}

func parseOsRelease(path string) (*DistributionInfo, error) {
	di := &DistributionInfo{ShortName: "linux"}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return di, fmt.Errorf("unable to obtain release info: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		entry := strings.Split(scanner.Text(), "=")
		switch entry[0] {
		case "":
			continue
		case "PRETTY_NAME":
			di.LongName = strings.Trim(entry[1], `"`)
		case "VERSION_ID":
			di.Version = strings.Trim(entry[1], `"`)
		case "ID":
			di.ShortName = strings.Trim(entry[1], `"`)
		}
		if di.LongName != "" && di.Version != "" && di.ShortName != "" {
			break
		}
	}

	return di, nil
}

func parseEnterpriseRelease(path string) (*DistributionInfo, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return &DistributionInfo{ShortName: "linux"}, fmt.Errorf("unable to obtain release info: %v", err)
	}
	rel := string(b)

	var sn string
	switch {
	case strings.Contains(rel, "CentOS"):
		sn = "centos"
	case strings.Contains(rel, "Red Hat"):
		sn = "rhel"
	case strings.Contains(rel, "Oracle"):
		sn = "ol"
	}

	return &DistributionInfo{
		ShortName: sn,
		LongName:  strings.Replace(rel, " release ", " ", 1),
		Version:   entRelVerRgx.FindString(rel),
	}, nil
}

func GetDistributionInfo() (*DistributionInfo, error) {
	var di *DistributionInfo
	var err error
	switch {
	// Check for /etc/os-release first.
	case exists(osRelease):
		di, err = parseOsRelease(osRelease)
	case exists(oRelease):
		di, err = parseEnterpriseRelease(oRelease)
	case exists(rhRelease):
		di, err = parseEnterpriseRelease(rhRelease)
	default:
		err = errors.New("unable to obtain release info, no known /etc/*-release exists")
	}
	if err != nil {
		return nil, err
	}

	out, err := exec.Command("/bin/uname", "-r").CombinedOutput()
	if err != nil {
		return nil, err
	}
	di.Kernel = strings.TrimSpace(string(out))
	return di, nil
}
