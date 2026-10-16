package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

// Fingerprint hashes the stable outcome of a build (D29): who moves, under
// which name, with which credentials and targets, and which conflicts were
// fixed how. Traffic, online times and derived status are left out: they
// change all the time and are taken fresh at run time.
func Fingerprint(r *Result) string {
	h := sha256.New()
	merge := int64(0)
	if r.Admin.Merge != nil {
		merge = r.Admin.Merge.ID
	}
	fmt.Fprintf(h, "admin|%s|%d|%d\n", r.Admin.Username, merge, r.Admin.RoleID)
	for _, c := range r.Conflicts {
		fmt.Fprintf(h, "job|%s|%s\n", c.Type, c.Severity)
	}
	items := slices.Clone(r.Items)
	slices.SortFunc(items, func(a, b *Item) int { return strings.Compare(a.Key, b.Key) })
	for _, it := range items {
		adopt := int64(0)
		if it.Adopt != nil {
			adopt = it.Adopt.ID
		}
		fmt.Fprintf(h, "user|%s|%t|%d|%s|%s\n", it.Key, it.Include, adopt, it.User.Username, strings.Join(it.Targets, ","))
		keys := it.User.Credentials.Keys()
		slices.Sort(keys)
		for _, k := range keys {
			sum := sha256.Sum256([]byte(k))
			fmt.Fprintf(h, "cred|%x\n", sum[:8])
		}
		for _, c := range sortedConflicts(append(slices.Clone(it.Resolved), it.Conflicts...)) {
			fmt.Fprintf(h, "conflict|%s|%s|%s\n", c.Type, c.Fix, c.Severity)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sortedConflicts(cs []Conflict) []Conflict {
	slices.SortFunc(cs, func(a, b Conflict) int {
		if c := strings.Compare(a.Type, b.Type); c != 0 {
			return c
		}
		return strings.Compare(a.Fix, b.Fix)
	})
	return cs
}
