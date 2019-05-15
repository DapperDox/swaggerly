/*
Copyright (C) 2016-2017 dapperdox.com

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/
package spec

import (
	"bufio"
	"os"
	"regexp"
	"strconv"

	"github.com/frinka/dapperdox/config"
	"github.com/frinka/dapperdox/logger"
)

var statusMapSplit = regexp.MustCompile(",")
var StatusCodes map[int]string

func LoadStatusCodes() {
	var statusfile string

	cfg, _ := config.Get()

	if len(cfg.AssetsDir) != 0 {
		statusfile = cfg.AssetsDir + "/status_codes.csv"
		logger.Tracef(nil, "Looking in assets dir for %s\n", statusfile)
		if _, err := os.Stat(statusfile); os.IsNotExist(err) {
			statusfile = ""
		}
	}
	if len(statusfile) == 0 && len(cfg.ThemeDir) != 0 {
		statusfile = cfg.ThemeDir + "/" + cfg.Theme + "/status_codes.csv"
		logger.Tracef(nil, "Looking in theme dir for %s\n", statusfile)
		if _, err := os.Stat(statusfile); os.IsNotExist(err) {
			statusfile = ""
		}
	}
	if len(statusfile) == 0 {
		statusfile = cfg.DefaultAssetsDir + "/themes/" + cfg.Theme + "/status_codes.csv"
		logger.Tracef(nil, "Looking in default theme dir for %s\n", statusfile)
		if _, err := os.Stat(statusfile); os.IsNotExist(err) {
			statusfile = ""
		}
	}
	if len(statusfile) == 0 {
		statusfile = cfg.DefaultAssetsDir + "/themes/default/status_codes.csv"
		logger.Tracef(nil, "Looking in default theme %s\n", statusfile)
		if _, err := os.Stat(statusfile); os.IsNotExist(err) {
			statusfile = ""
		}
	}

	if len(statusfile) == 0 {
		logger.Tracef(nil, "No status code map file found.")
		return
	}
	logger.Tracef(nil, "Processing HTTP status code file: %s\n", statusfile)
	file, err := os.Open(statusfile)

	if err != nil {
		logger.Errorf(nil, "Error: %s", err)
		return
	}
	defer file.Close()

	StatusCodes = make(map[int]string)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		indexes := statusMapSplit.FindStringIndex(line)
		if indexes == nil {
			return
		}
		i, err := strconv.Atoi(line[0 : indexes[1]-1])
		if err != nil {
			logger.Errorf(nil, "Invalid HTTP status code in csv file: '%s'\n", line)
			continue
		}
		status := i
		desc := line[indexes[1]:]

		StatusCodes[status] = string(desc)
	}

	if err := scanner.Err(); err != nil {
		logger.Errorf(nil, "Error: %s", err)
	}
}

func HTTPStatusDescription(status int) string {
	if desc, ok := StatusCodes[status]; ok {
		return desc
	}
	return ""
}
