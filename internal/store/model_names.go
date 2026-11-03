package store

import (
	"fmt"
	"log"
	"sort"
)

// modelColumns lists every column that stores a configured model name.
var modelColumns = []struct{ table, column string }{
	{"conversation", "model"},
	{"character", "model"},
	{"completion", "model"},
	{"research_job", "model"},
	{"research_job", "html_report_model"},
	{"research_job", "worker_model"},
	{"research_remix", "model"},
	{"research_report", "model"},
}

// ModelUsage is how many rows reference one model name, per table.column.
type ModelUsage struct {
	Name   string
	Counts map[string]int
	Total  int
}

// ModelNamesInUse returns every model name stored in the database with its
// reference counts, sorted by name.
func (s *Store) ModelNamesInUse() ([]ModelUsage, error) {
	byName := map[string]*ModelUsage{}
	for _, mc := range modelColumns {
		rows, err := s.db.Query(fmt.Sprintf(
			`SELECT %[2]s, COUNT(*) FROM %[1]s WHERE %[2]s IS NOT NULL AND %[2]s != '' GROUP BY %[2]s`, mc.table, mc.column))
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var name string
			var n int
			if err := rows.Scan(&name, &n); err != nil {
				rows.Close()
				return nil, err
			}
			u := byName[name]
			if u == nil {
				u = &ModelUsage{Name: name, Counts: map[string]int{}}
				byName[name] = u
			}
			u.Counts[mc.table+"."+mc.column] += n
			u.Total += n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	out := make([]ModelUsage, 0, len(byName))
	for _, u := range byName {
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// RenameModel replaces model name from with to in every column that stores a
// model name, in one transaction, and returns the number of rows changed.
// Research report labels that combine two models ("md / html") are display
// text and are left alone.
func (s *Store) RenameModel(from, to string) (int64, error) {
	log.Printf("store: RenameModel — renaming %q → %q", from, to)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var total int64
	for _, mc := range modelColumns {
		res, err := tx.Exec(fmt.Sprintf(`UPDATE %s SET %s = ? WHERE %s = ?`, mc.table, mc.column, mc.column), to, from)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			log.Printf("store:   %s.%s — %d row(s)", mc.table, mc.column, n)
		}
		total += n
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	log.Printf("store: RenameModel — renamed %q → %q in %d row(s)", from, to, total)
	return total, nil
}
