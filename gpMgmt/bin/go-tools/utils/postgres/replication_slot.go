package postgres

import (
	"fmt"
	"strconv"

	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gp-common-go-libs/gplog"
)

var newDBConnFromEnvironment = dbconn.NewDBConnFromEnvironment

type ReplicationSlot struct {
	Host string
	Port int
	Name string
}

func NewReplicationSlot(host string, port int, name string) ReplicationSlot {
	return ReplicationSlot{
		Host: host,
		Port: port,
		Name: name,
	}
}

func (r *ReplicationSlot) getDbConn() (*dbconn.DBConn, error) {
	conn := newDBConnFromEnvironment("postgres")
	conn.Host = r.Host
	conn.Port = r.Port

	err := conn.Connect(1, true)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (r *ReplicationSlot) SlotExists() (bool, error) {
	gplog.Info("Checking if replication slot %s exists on %s:%d", r.Name, r.Host, r.Port)
	conn, err := r.getDbConn()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	query := fmt.Sprintf("SELECT count(*) FROM pg_catalog.pg_replication_slots WHERE slot_name = '%s'", r.Name)
	gplog.Debug("executing query %s", query)
	result, err := dbconn.SelectString(conn, query)
	if err != nil {
		return false, err
	}

	count, err := strconv.Atoi(result)
	if err != nil {
		return false, err
	}

	if count > 0 {
		gplog.Info("Replication slot %s exists on %s:%d", r.Name, r.Host, r.Port)
		return true, nil
	}

	gplog.Info("Replication slot %s does not exist on %s:%d", r.Name, r.Host, r.Port)
	return false, nil
}

func (r *ReplicationSlot) DropSlot() error {
	gplog.Info("Dropping replication slot %s on %s:%d", r.Name, r.Host, r.Port)
	conn, err := r.getDbConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	query := fmt.Sprintf("SELECT pg_drop_replication_slot('%s')", r.Name)
	gplog.Debug("executing query %s", query)
	_, err = conn.Exec(query)
	if err != nil {
		return err
	}

	gplog.Info("Successfully dropped replication slot %s on %s:%d", r.Name, r.Host, r.Port)
	return nil
}

func DropSlotIfExists(host string, port int, name string) error {
	slot := NewReplicationSlot(host, port, name)

	exists, err := slot.SlotExists()
	if err != nil {
		return err
	}

	if exists {
		err := slot.DropSlot()
		if err != nil {
			return err
		}
	}

	return nil
}

// used only for testing
func SetNewDBConnFromEnvironment(customFunc func(dbname string) *dbconn.DBConn) {
	newDBConnFromEnvironment = customFunc
}

func ResetNewDBConnFromEnvironment() {
	newDBConnFromEnvironment = dbconn.NewDBConnFromEnvironment
}
