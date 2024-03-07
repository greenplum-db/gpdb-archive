package postgres

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpdb/gp/utils"
)

const (
	postgresqlConfFile       = "postgresql.conf"
	postgresInternalConfFile = "internal.auto.conf"
)

// UpdatePostgresqlConf updates given config params to postgresql.conf file
func UpdatePostgresqlConf(pgdata string, configParams map[string]string, overwrite bool) error {
	gplog.Debug("Updating %s for data directory %s with: %s", postgresqlConfFile, pgdata, configParams)
	err := updateConfFile(postgresqlConfFile, pgdata, configParams, overwrite)
	if err != nil {
		return err
	}

	gplog.Info("Successfully updated %s for data directory %s", postgresqlConfFile, pgdata)
	return nil
}

func UpdatePostgresInternalConf(pgdata string, dbid int) error {
	postgresInternalConfFilePath := filepath.Join(pgdata, postgresInternalConfFile)
	file, err := utils.System.OpenFile(postgresInternalConfFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	contents := fmt.Sprintf("gp_dbid = %d", dbid)
	_, err = file.WriteString(contents)
	if err != nil {
		return err
	}

	gplog.Info("Successfully created %s for data directory %s", postgresInternalConfFile, pgdata)
	return nil
}

// updateConfFile updates postresql.conf file with the provided config params.
// If the config exists, comments the existing line with a # and adds a new line.
func updateConfFile(filename, pgdata string, configParams map[string]string, overwrite bool) error {
	var line string
	confFilePath := filepath.Join(pgdata, filename)

	file, err := utils.System.Open(confFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	updatedLines := []string{}
	updatedKeys := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line = scanner.Text()

		if strings.HasPrefix(line, "#") {
			// Skip the commented line
			updatedLines = append(updatedLines, line)
			continue
		}

		for key, value := range configParams {
			pattern, err := regexp.Compile(fmt.Sprintf("^%s[\\s=]+", key))
			if err != nil {
				return err
			}

			if pattern.MatchString(line) {
				if !overwrite {
					line = fmt.Sprintf("%s = %s # %s", key, quoteIfString(value), line) // comment the previous entry
				} else {
					line = fmt.Sprintf("%s = %s", key, quoteIfString(value))
				}

				// Mark the key as updated to delete later from configParams
				updatedKeys = append(updatedKeys, key)
				break
			}
		}

		updatedLines = append(updatedLines, line)
	}

	// Delete the updated keys to avoid duplicated entries
	for _, key := range updatedKeys {
		delete(configParams, key)
	}

	// Add the remaining entries
	for key, value := range configParams {
		line := fmt.Sprintf("%s = %s", key, quoteIfString(value))
		updatedLines = append(updatedLines, line)
	}

	err = utils.WriteLinesToFile(confFilePath, updatedLines)
	if err != nil {
		return err
	}

	return nil
}

// quoteIfString encloses string values inside quotes.
// Checks if string is a number, then skips adding quotes to the string.
func quoteIfString(value string) string {
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return value
	} else {
		return fmt.Sprintf("'%s'", value)
	}
}

// GetConfigValue retrieves the value of a configuration parameter from the PostgreSQL configuration file.
// It takes the path to the PostgreSQL data directory (pgdata) and the name of the configuration parameter (config) as input.
// It returns the value of the configuration parameter as a string, and in case if same parameter is present multiple
// times, it will return the value of the last one
func GetConfigValue(pgdata, config string) (string, error) {
	var value string
	postgresqlConfFilePath := filepath.Join(pgdata, postgresqlConfFile)

	file, err := utils.System.Open(postgresqlConfFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.ReplaceAll(strings.TrimSpace(scanner.Text()), "=", " ")

		if strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		if parts[0] == config {
			value = parts[1]
		}
	}

	if value == "" {
		return value, fmt.Errorf("did not find any config parameter named %q in %s", config, postgresqlConfFilePath)
	}

	return strings.Trim(value, "'"), nil
}
