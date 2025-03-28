package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/schema"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
)

// TakeSnapshot Create a snapshot of the model
func (mod *Model) TakeSnapshot(inMemory bool) (string, error) {
	name := mod.generateSnapshotName()
	err := mod.createSnapshotTable(name, inMemory)
	if err != nil {
		return "", fmt.Errorf("create snapshot table failed: %s", err)
	}

	if capsule.Global == nil {
		return "", fmt.Errorf("capsule is not initialized")
	}

	// Copy the data
	qb := capsule.Global.Query()
	sql := fmt.Sprintf("INSERT INTO `%s` SELECT * FROM `%s`", name, mod.MetaData.Table.Name)
	_, err = qb.ExecWrite(sql)
	if err != nil {
		// Drop the snapshot table
		dropErr := mod.DropSnapshotTable(name)
		if dropErr != nil {
			color.Red("[TakeSnapshot] drop snapshot table failed: %s", dropErr)
			log.Error("[TakeSnapshot] drop snapshot table failed: %s", dropErr)
		}

		color.Red("[TakeSnapshot] create snapshot table failed: %s", err)
		log.Error("[TakeSnapshot] create snapshot table failed: %s", err)
		return "", err
	}
	return name, nil
}

// SnapshotExists Check if the snapshot table exists
func (mod *Model) SnapshotExists(name string) (bool, error) {
	// Select the connector
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}
	sch := schema.Use(connector)
	exists, err := sch.TableExists(name)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// RestoreSnapshotByRename Restore the model from the snapshot by renaming the snapshot table
func (mod *Model) RestoreSnapshotByRename(name string) error {
	// Check if the snapshot table exists
	exists, err := mod.SnapshotExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("failed to restore snapshot, snapshot table not found: %s", name)
	}

	// Select the connector
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	sch := schema.Use(mod.MetaData.Connector)
	exists, err = sch.TableExists(mod.MetaData.Table.Name)
	if err != nil {
		return err
	}
	if exists {
		err = sch.TableDrop(mod.MetaData.Table.Name) // Drop the table
		if err != nil {
			color.Red("[RestoreSnapshot] drop table failed: %s", err)
			log.Error("[RestoreSnapshot] drop table failed: %s", err)
			return err
		}
	}

	// Rename the snapshot table
	err = sch.TableRename(name, mod.MetaData.Table.Name)
	if err != nil {
		return err
	}

	return nil
}

// RestoreSnapshot Restore the model from the snapshot
func (mod *Model) RestoreSnapshot(name string) error {
	// Check if the snapshot table exists
	exists, err := mod.SnapshotExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("failed to restore snapshot, snapshot table not found: %s", name)
	}

	// Select the connector
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	sch := schema.Use(mod.MetaData.Connector)
	exists, err = sch.TableExists(mod.MetaData.Table.Name)
	if err != nil {
		return err
	}
	if exists {
		err = sch.TableDrop(mod.MetaData.Table.Name) // Drop the table
		if err != nil {
			color.Red("[RestoreSnapshot] drop table failed: %s", err)
			log.Error("[RestoreSnapshot] drop table failed: %s", err)
			return err
		}
	}

	// Recreate the table
	blueprint, err := mod.Blueprint()
	if err != nil {
		color.Red("[RestoreSnapshot] recreate table failed: %s", err)
		log.Error("[RestoreSnapshot] recreate table failed: %s", err)
		return err
	}
	err = sch.TableCreate(mod.MetaData.Table.Name, blueprint)
	if err != nil {
		color.Red("[RestoreSnapshot] recreate table failed: %s", err)
		log.Error("[RestoreSnapshot] recreate table failed: %s", err)
		return err
	}

	// Restore the snapshot
	qb := capsule.Global.Query()
	sql := fmt.Sprintf("INSERT INTO `%s` SELECT * FROM `%s`", mod.MetaData.Table.Name, name)
	_, err = qb.ExecWrite(sql)
	if err != nil {
		color.Red("[RestoreSnapshot] restore snapshot failed: %s", err)
		log.Error("[RestoreSnapshot] restore snapshot failed: %s", err)
		return err
	}

	return nil
}

// DropSnapshotTable Drop the snapshot table
func (mod *Model) DropSnapshotTable(name string) error {

	prefix := fmt.Sprintf("%s_snapshot_", mod.MetaData.Table.Name)
	if !strings.HasPrefix(name, prefix) {
		return fmt.Errorf("failed to drop snapshot table, invalid name: %s", name)
	}

	// Select the connector
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	sch := schema.Use(mod.MetaData.Connector)
	exists, err := sch.TableExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return sch.TableDrop(name)
}

func (mod *Model) createSnapshotTable(name string, inMemory bool) error {
	// Create table
	blueprint, err := mod.Blueprint()
	if err != nil {
		return err
	}

	// Select the connector
	connector := mod.MetaData.Connector
	if connector == "" {
		connector = "default"
	}

	sch := schema.Use(connector)
	if inMemory {
		blueprint.Temporary = true
	}

	return sch.TableCreate(name, blueprint)
}

// generateSnapshotName generate a snapshot name
func (mod *Model) generateSnapshotName() string {
	return fmt.Sprintf("%s_snapshot_%s", mod.MetaData.Table.Name, time.Now().Format("20060102150405_000"))
}
