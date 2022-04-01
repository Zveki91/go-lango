package services

import (
	"bytes"
	"fmt"
	"github.com/jackc/pgx"
	"log"
	"strings"
	"text/template"
)

const (
	minPageSize     = 1
	defaultPageSize = 10
	maxPageSize     = 20
)

var queriesCache = make(map[string]*template.Template)

func isUniqueViolation(err error) bool {
	pgErr, ok := err.(pgx.PgError)
	return ok && pgErr.Code == "..."
}

func queryBuilder(text string, data map[string]interface{}) (string, []interface{}, error) {
	t, ok := queriesCache[text]
	if !ok {
		var err error
		t, err = template.New("query").Parse(text)
		if err != nil {
			return "", nil, fmt.Errorf("could not parse query template")
		}

		queriesCache[text] = t
	}

	var wr bytes.Buffer
	if err := t.Execute(&wr, data); err != nil {
		return "", nil, fmt.Errorf("could not execute query")
	}

	query := wr.String()
	args := []interface{}{}
	for key, val := range data {
		if !strings.Contains(query, "@"+key) {
			continue
		}
		args = append(args, val)
		query = strings.ReplaceAll(query, "@"+key, fmt.Sprintf("$%d", len(args)))
	}
	log.Println(query)
	return query, args, nil
}

func normalizePageSize(i int) int {
	if i == 0 {
		return defaultPageSize
	}
	if i < minPageSize {
		return minPageSize
	}
	if i > maxPageSize {
		return maxPageSize
	}
	return i
}
