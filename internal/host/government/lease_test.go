package government

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func invalidLeaseRecords(t *testing.T, valid []byte) map[string][]byte {
	t.Helper()
	var original map[string]any
	if err := json.Unmarshal(valid, &original); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{}
	mutate := func(name, field string, value any, remove bool) {
		t.Helper()
		copy := map[string]any{}
		for k, v := range original {
			copy[k] = v
		}
		if remove {
			delete(copy, field)
		} else {
			copy[field] = value
		}
		data, err := json.Marshal(copy)
		if err != nil {
			t.Fatal(err)
		}
		cases[name] = data
	}
	for _, field := range []string{"version", "host", "user", "pid", "token", "createdAt"} {
		mutate("missing-"+field, field, nil, true)
		mutate("null-"+field, field, nil, false)
	}
	for name, value := range map[string]any{"zero": 0, "negative": -1, "string": "42", "fractional": 1.5} {
		mutate("pid-"+name, "pid", value, false)
	}
	for name, value := range map[string]string{"empty": "", "short": "a", "long": strings.Repeat("a", 66), "nonhex": strings.Repeat("z", 64), "uppercase": strings.Repeat("A", 64)} {
		mutate("token-"+name, "token", value, false)
	}
	for name, value := range map[string]string{"zero": "0001-01-01T00:00:00Z", "invalid": "yesterday", "offset": "2026-10-07T22:00:00+02:00", "noncanonical": "2026-10-07T20:00:00.000Z"} {
		mutate("time-"+name, "createdAt", value, false)
	}
	for _, field := range []string{"host", "user"} {
		mutate(field+"-blank", field, " ", false)
		mutate(field+"-untrimmed", field, " "+original[field].(string), false)
		mutate(field+"-control", field, original[field].(string)+"\x00", false)
	}
	mutate("unsupported-version", "version", 2, false)
	mutate("unknown-field", "extra", true, false)
	mutate("case-alias", "Token", original["token"], false)
	body := bytes.TrimSpace(valid)
	cases["duplicate"] = append(append([]byte{}, body[:len(body)-1]...), []byte(`,"pid":1}`)...)
	cases["trailing"] = append(append([]byte{}, body...), []byte(` {}`)...)
	cases["invalid-utf8"] = append(append([]byte{}, body...), 0xff)
	cases["oversize"] = []byte(strings.Repeat(" ", 8193))
	return cases
}

func TestAcquireLeaseRejectsIncompleteOrCorruptRecordsWithoutChangingBytes(t *testing.T) {
	seedPath := filepath.Join(t.TempDir(), "seed.lease")
	seed, err := AcquireLease(seedPath)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	cases := invalidLeaseRecords(t, valid)
	var owner leaseOwner
	if err := json.Unmarshal(valid, &owner); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"host", "user"} {
		foreign := owner
		if field == "host" {
			foreign.Host += "-foreign"
		} else {
			foreign.User += "-foreign"
		}
		cases["foreign-"+field], err = json.Marshal(foreign)
		if err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "writer.lease")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if lease, err := AcquireLease(path); err == nil {
				_ = lease.Close()
				t.Fatal("unrecognized or foreign record was taken over")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, data) {
				t.Fatalf("rejected record changed: read error=%v", err)
			}
			// Rejection must release OS exclusion, so a recognized restored
			// record can still be legitimately acquired and verified.
			if err := os.WriteFile(path, valid, 0600); err != nil {
				t.Fatal(err)
			}
			lease, err := AcquireLease(path)
			if err != nil {
				t.Fatalf("valid released record could not resume: %v", err)
			}
			defer lease.Close()
			if lease.Token() == owner.Token || lease.Verify() != nil {
				t.Fatal("resumed lease did not receive a valid fresh fencing record")
			}
		})
	}
}

func TestLeaseVerifyRejectsCorruptOrChangedOwnerRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writer.lease")
	lease, err := AcquireLease(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := invalidLeaseRecords(t, valid)
	changed := lease.owner
	changed.CreatedAt = changed.CreatedAt.Add(time.Nanosecond)
	cases["changed-valid-createdAt"], err = json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := lease.Verify(); err == nil {
				t.Fatal("Verify accepted an invalid or changed owner record")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, data) {
				t.Fatalf("Verify changed the rejected record: %v", err)
			}
		})
	}
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	if err := lease.Verify(); err != nil {
		t.Fatalf("original held record no longer verifies: %v", err)
	}
}
