package pgtune

import "testing"

func TestGetSettingsGroup(t *testing.T) {
	okLabels := []string{MemoryLabel, ParallelLabel, WALLabel, MiscLabel}
	mem := uint64(1024)
	cpus := 4
	pgVersion := "10"
	for _, label := range okLabels {
		sg := GetSettingsGroup(label, pgVersion, mem, cpus)
		if sg == nil {
			t.Errorf("settings group unexpectedly nil for label %s", label)
		}
		switch x := sg.(type) {
		case *MemorySettingsGroup:
			if x.totalMemory != mem || x.cpus != cpus {
				t.Errorf("memory settings group incorrect: got %d,%d want %d,%d", x.totalMemory, x.cpus, mem, cpus)
			}
		case *ParallelSettingsGroup:
			if x.cpus != cpus {
				t.Errorf("parallel settings group incorrect: got %d want %d", x.cpus, cpus)
			}
			if x.pgVersion != pgVersion {
				t.Errorf("parallel settings group incorrect: got %s want %s", x.pgVersion, pgVersion)
			}
		case *WALSettingsGroup:
			if x.totalMemory != mem {
				t.Errorf("WAL settings group incorrect: got %d want %d", x.totalMemory, mem)
			}
		case *MiscSettingsGroup:
		default:
			t.Errorf("unexpected type for settings group %T", x)
		}
	}

	// this should panic on unknown label
	func() {
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		GetSettingsGroup("foo", pgVersion, mem, cpus)
	}()
}
