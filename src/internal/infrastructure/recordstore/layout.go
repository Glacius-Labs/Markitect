package recordstore

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

const RootPath = ".markitect/records"

type Kind string

const (
	Run         Kind = "run"
	Exploration Kind = "exploration"
	Briefing    Kind = "briefing"
	Brownfield  Kind = "brownfield"
)

// Name identifies a closed operational role, never an arbitrary repository file.
type Name struct {
	Kind     Kind
	ID, Role string
}

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)
var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)
var revisionName = regexp.MustCompile(`^[0-9]{8,}\.json$`)
var component = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func safeComponent(s string) bool {
	if !component.MatchString(s) || strings.HasSuffix(s, ".") || len(s) > 255 {
		return false
	}
	base := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return false
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return false
	}
	return true
}
func validKind(k Kind) bool { return k == Run || k == Exploration || k == Briefing || k == Brownfield }
func validID(k Kind, id string) bool {
	switch k {
	case Run:
		return hex32.MatchString(id) || strings.HasPrefix(id, "full-") && hex64.MatchString(strings.TrimPrefix(id, "full-"))
	case Briefing:
		return id == "accepted"
	default:
		return safeComponent(id)
	}
}
func (n Name) Path() (string, error) {
	if !validKind(n.Kind) {
		return "", errors.New("unknown record kind")
	}
	if n.ID == "" && n.Kind == Run && n.Role == ".write.lock" {
		return RootPath + "/run/.write.lock", nil
	}
	if !validID(n.Kind, n.ID) {
		return "", fmt.Errorf("invalid %s record ID", n.Kind)
	}
	valid := false
	if n.Kind != Run || strings.HasPrefix(n.ID, "full-") {
		valid = n.Role == "record.json"
	}
	switch n.Kind {
	case Brownfield:
		valid = valid || n.Role == "session.lock" || n.Role == "manager-runs.lock"
	case Briefing:
		valid = valid || n.Role == ".briefings.lock"
	case Run:
		if hex32.MatchString(n.ID) {
			valid = n.Role == "plan.json"
			parts := strings.Split(n.Role, "/")
			if len(parts) == 2 {
				switch parts[0] {
				case "states":
					valid = revisionName.MatchString(parts[1]) && strings.TrimSuffix(parts[1], ".json") != "00000000"
				case "candidates", "verification", "apply":
					valid = hex32.MatchString(strings.TrimSuffix(parts[1], ".json")) && strings.HasSuffix(parts[1], ".json")
				case "reports":
					valid = safeComponent(parts[1]) && strings.HasSuffix(parts[1], ".json")
				}
			}
			if len(parts) == 3 && parts[0] == "external" && (parts[1] == "packets" || parts[1] == "responses") {
				valid = hex32.MatchString(strings.TrimSuffix(parts[2], ".json")) && strings.HasSuffix(parts[2], ".json")
			}
		}
	}
	if !valid || path.Clean(n.Role) != n.Role {
		return "", errors.New("unsupported record role")
	}
	return RootPath + "/" + string(n.Kind) + "/" + n.ID + "/" + n.Role, nil
}
func Parse(relative string) (Name, error) {
	if !strings.HasPrefix(relative, RootPath+"/") || strings.ContainsAny(relative, "\\:\x00") || path.Clean(relative) != relative {
		return Name{}, errors.New("invalid operational record path")
	}
	parts := strings.Split(strings.TrimPrefix(relative, RootPath+"/"), "/")
	if len(parts) == 2 && parts[0] == "run" && parts[1] == ".write.lock" {
		return Name{Kind: Run, Role: ".write.lock"}, nil
	}
	if len(parts) < 3 {
		return Name{}, errors.New("record path requires kind, ID and role")
	}
	n := Name{Kind: Kind(parts[0]), ID: parts[1], Role: strings.Join(parts[2:], "/")}
	p, err := n.Path()
	if err != nil || p != relative {
		return Name{}, errors.New("invalid closed record path")
	}
	return n, nil
}
func directoryPath(kind Kind, id, role string) (string, error) {
	if !validKind(kind) {
		return "", errors.New("unknown record kind")
	}
	p := RootPath + "/" + string(kind)
	if id == "" {
		if role != "" {
			return "", errors.New("directory role requires ID")
		}
		return p, nil
	}
	if !validID(kind, id) {
		return "", errors.New("invalid record directory ID")
	}
	p += "/" + id
	if role == "" {
		return p, nil
	}
	if kind != Run || !hex32.MatchString(id) {
		return "", errors.New("unsupported record directory")
	}
	switch role {
	case "states", "candidates", "reports", "verification", "apply", "external", "external/packets", "external/responses":
		return p + "/" + role, nil
	}
	return "", errors.New("unsupported record directory role")
}
