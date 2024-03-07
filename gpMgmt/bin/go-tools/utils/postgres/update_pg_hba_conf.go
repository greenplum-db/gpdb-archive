package postgres

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpdb/gp/utils"
)

const pgHbaConfFile = "pg_hba.conf"

func BuildCoordinatorPgHbaConf(pgdata string, addrs []string) error {
	pgHbaFilePath := filepath.Join(pgdata, pgHbaConfFile)

	file, err := utils.System.Open(pgHbaFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	updatedLines := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") {
			updatedLines = append(updatedLines, line)
		}
	}

	user, err := utils.System.CurrentUser()
	if err != nil {
		return err
	}

	localAccessString := "local\t%s\t%s\tident"

	// Add local access entries
	line := fmt.Sprintf(localAccessString, "all", user.Username)
	updatedLines = append(updatedLines, line)
	line = fmt.Sprintf(localAccessString, "replication", user.Username)
	updatedLines = append(updatedLines, line)

	entries := createPgHbaEntries(append([]string{"localhost"}, addrs...), user.Username, true)
	updatedLines = append(updatedLines, entries...)

	err = utils.WriteLinesToFile(pgHbaFilePath, updatedLines)
	if err != nil {
		return err
	}

	gplog.Info("Successfully updated %s for data directory %s", pgHbaConfFile, pgdata)
	return nil
}

func UpdateSegmentPgHbaConf(pgdata string, addrs []string, replication bool, coordinatorAddrs ...string) error {
	gplog.Info("Starting to update %s for data directory %s", pgHbaConfFile, pgdata)
	var entries []string

	if len(coordinatorAddrs) > 0 {
		entries = append(entries, createPgHbaEntries(coordinatorAddrs, "all", false)...)
	}

	user, err := utils.System.CurrentUser()
	if err != nil {
		return err
	}

	entries = append(entries, createPgHbaEntries(addrs, user.Username, replication)...)
	err = appendPgHbaEntries(pgdata, entries)
	if err != nil {
		return err
	}

	gplog.Info("Successfully updated %s for data directory %s", pgHbaConfFile, pgdata)
	return nil
}

func appendPgHbaEntries(pgdata string, entries []string) error {
	pgHbaConfFilePath := filepath.Join(pgdata, pgHbaConfFile)

	content, err := utils.System.ReadFile(pgHbaConfFilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	lines = append(lines, entries...)
	lines = removeDuplicates(lines)

	err = utils.WriteLinesToFile(pgHbaConfFilePath, lines)
	if err != nil {
		return err
	}

	return nil
}

func createPgHbaEntries(addrs []string, username string, replication bool) []string {
	var entries []string
	entryString := "host\t%s\t%s\t%s\ttrust"

	for _, addr := range addrs {
		entries = append(entries, fmt.Sprintf(entryString, "all", username, addr))
	}

	if replication {
		addrs = append([]string{"samehost"}, addrs...)
		for _, addr := range addrs {
			entries = append(entries, fmt.Sprintf(entryString, "replication", username, addr))
		}
	}

	return entries
}

func reFormatEntry(line string) string {
	return strings.Join(strings.Fields(line), "\t")
}

func removeDuplicates(entries []string) []string {
	var result []string
	existingEntries := make(map[string]bool)

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)

		if strings.HasPrefix(entry, "#") {
			existingEntries[entry] = true
			result = append(result, entry)
			continue
		}

		entry = reFormatEntry(entry)
		if _, ok := existingEntries[entry]; !ok {
			existingEntries[entry] = true
			result = append(result, entry)
		}
	}

	return result
}
