package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRebind(t *testing.T) {
	query := "SELECT * FROM users WHERE id = ? AND username = ? AND note != 'is it ?'"

	// SQLite & MySQL should remain unchanged
	assert.Equal(t, query, Rebind("sqlite", query))
	assert.Equal(t, query, Rebind("mysql", query))

	// Postgres should convert ? outside quotes to $1, $2
	expectedPostgres := "SELECT * FROM users WHERE id = $1 AND username = $2 AND note != 'is it ?'"
	assert.Equal(t, expectedPostgres, Rebind("postgres", query))
}
