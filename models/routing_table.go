package models

import "reflect"

type RoutingKey struct {
	Port uint16
}

type BackendServerInfo struct {
	Address string
	Port    uint16
}

type BackendServerInfos []BackendServerInfo

type RoutingTableEntry struct {
	Backends map[BackendServerInfo]struct{}
}

func NewRoutingTableEntry(backends BackendServerInfos) RoutingTableEntry {
	routingTableEntry := RoutingTableEntry{
		Backends: make(map[BackendServerInfo]struct{}),
	}
	for _, backend := range backends {
		routingTableEntry.Backends[backend] = struct{}{}
	}
	return routingTableEntry
}

type RoutingTable struct {
	Entries map[RoutingKey]RoutingTableEntry
}

func NewRoutingTable() RoutingTable {
	return RoutingTable{
		Entries: make(map[RoutingKey]RoutingTableEntry),
	}
}

func (table RoutingTable) Set(key RoutingKey, newEntry RoutingTableEntry) bool {
	existingEntry, ok := table.Entries[key]
	if ok == true && reflect.DeepEqual(existingEntry, newEntry) {
		return false
	}
	table.Entries[key] = newEntry
	return true
}

func (table RoutingTable) UpsertBackendServerInfo(key RoutingKey, backendServerInfo BackendServerInfo) bool {
	existingEntry, ok := table.Entries[key]
	updated := false
	if ok == false {
		existingEntry = NewRoutingTableEntry(BackendServerInfos{})
		table.Entries[key] = existingEntry
	}
	_, backendFound := existingEntry.Backends[backendServerInfo]
	if backendFound == false {
		existingEntry.Backends[backendServerInfo] = struct{}{}
		updated = true
	}
	return updated
}

func (table RoutingTable) DeleteBackendServerInfo(key RoutingKey, backendServerInfo BackendServerInfo) bool {
	existingEntry, ok := table.Entries[key]
	deleted := false
	if ok == true {
		_, backendFound := existingEntry.Backends[backendServerInfo]
		if backendFound == true {
			delete(existingEntry.Backends, backendServerInfo)
			deleted = true
		}
	}
	return deleted
}

func (table RoutingTable) Get(key RoutingKey) RoutingTableEntry {
	return table.Entries[key]
}

func (table RoutingTable) Size() int {
	return len(table.Entries)
}
