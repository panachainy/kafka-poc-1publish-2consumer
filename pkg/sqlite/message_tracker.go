package sqlite

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// MessageTracker handles SQLite operations for tracking seen messages and retry counts
type MessageTracker struct {
	db *sql.DB
}

// NewMessageTracker creates a new message tracker with SQLite database
func NewMessageTracker(dbPath string) (*MessageTracker, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	tracker := &MessageTracker{db: db}
	if err := tracker.createTables(); err != nil {
		return nil, err
	}

	return tracker, nil
}

// createTables creates the necessary tables if they don't exist
func (mt *MessageTracker) createTables() error {
	// Table for tracking seen messages
	seenTableSQL := `
	CREATE TABLE IF NOT EXISTS seen_messages (
		unique_id TEXT PRIMARY KEY,
		consumer_group TEXT NOT NULL,
		first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Table for tracking retry counts
	retryTableSQL := `
	CREATE TABLE IF NOT EXISTS retry_counts (
		unique_id TEXT NOT NULL,
		consumer_group TEXT NOT NULL,
		retry_count INTEGER DEFAULT 0,
		last_retry DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (unique_id, consumer_group)
	);`

	// Create indexes for better performance
	indexSQL := `
	CREATE INDEX IF NOT EXISTS idx_seen_consumer_group ON seen_messages(consumer_group);
	CREATE INDEX IF NOT EXISTS idx_retry_consumer_group ON retry_counts(consumer_group);
	`

	queries := []string{seenTableSQL, retryTableSQL, indexSQL}
	for _, query := range queries {
		if _, err := mt.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// IsSeen checks if a message has been seen before
func (mt *MessageTracker) IsSeen(uniqueID, consumerGroup string) (bool, error) {
	query := `SELECT 1 FROM seen_messages WHERE unique_id = ? AND consumer_group = ?`
	var exists int
	err := mt.db.QueryRow(query, uniqueID, consumerGroup).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MarkAsSeen marks a message as seen
func (mt *MessageTracker) MarkAsSeen(uniqueID, consumerGroup string) error {
	query := `
	INSERT INTO seen_messages (unique_id, consumer_group, first_seen, last_seen)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(unique_id) DO UPDATE SET
		last_seen = CURRENT_TIMESTAMP`

	now := time.Now()
	_, err := mt.db.Exec(query, uniqueID, consumerGroup, now, now)
	return err
}

// GetRetryCount gets the current retry count for a message
func (mt *MessageTracker) GetRetryCount(uniqueID, consumerGroup string) (int, error) {
	query := `SELECT retry_count FROM retry_counts WHERE unique_id = ? AND consumer_group = ?`
	var retryCount int
	err := mt.db.QueryRow(query, uniqueID, consumerGroup).Scan(&retryCount)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return retryCount, nil
}

// IncrementRetryCount increments the retry count for a message
func (mt *MessageTracker) IncrementRetryCount(uniqueID, consumerGroup string) (int, error) {
	// First, try to increment existing record
	updateQuery := `
	UPDATE retry_counts
	SET retry_count = retry_count + 1, last_retry = CURRENT_TIMESTAMP
	WHERE unique_id = ? AND consumer_group = ?`

	result, err := mt.db.Exec(updateQuery, uniqueID, consumerGroup)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	// If no rows were affected, insert a new record with retry_count = 1
	if rowsAffected == 0 {
		insertQuery := `
		INSERT INTO retry_counts (unique_id, consumer_group, retry_count, last_retry)
		VALUES (?, ?, 1, CURRENT_TIMESTAMP)`

		_, err = mt.db.Exec(insertQuery, uniqueID, consumerGroup)
		if err != nil {
			return 0, err
		}
		return 1, nil
	}

	// Get the updated retry count
	return mt.GetRetryCount(uniqueID, consumerGroup)
}

// DeleteRetryCount removes the retry count record for a message (called after successful processing)
func (mt *MessageTracker) DeleteRetryCount(uniqueID, consumerGroup string) error {
	query := `DELETE FROM retry_counts WHERE unique_id = ? AND consumer_group = ?`
	_, err := mt.db.Exec(query, uniqueID, consumerGroup)
	return err
}

// CleanupOldRecords removes old records to prevent database growth
func (mt *MessageTracker) CleanupOldRecords(olderThanDays int) error {
	// Clean up seen messages older than specified days
	seenCleanupQuery := `
	DELETE FROM seen_messages
	WHERE last_seen < datetime('now', '-' || ? || ' days')`

	// Clean up retry counts for messages not retried in specified days
	retryCleanupQuery := `
	DELETE FROM retry_counts
	WHERE last_retry < datetime('now', '-' || ? || ' days')`

	queries := []string{seenCleanupQuery, retryCleanupQuery}
	for _, query := range queries {
		if _, err := mt.db.Exec(query, olderThanDays); err != nil {
			log.Printf("Error cleaning up old records: %v", err)
			return err
		}
	}

	log.Printf("Cleaned up records older than %d days", olderThanDays)
	return nil
}

// GetStats returns statistics about the database
func (mt *MessageTracker) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	// Count seen messages
	var seenCount int
	err := mt.db.QueryRow("SELECT COUNT(*) FROM seen_messages").Scan(&seenCount)
	if err != nil {
		return nil, err
	}
	stats["seen_messages"] = seenCount

	// Count retry records
	var retryCount int
	err = mt.db.QueryRow("SELECT COUNT(*) FROM retry_counts").Scan(&retryCount)
	if err != nil {
		return nil, err
	}
	stats["retry_records"] = retryCount

	return stats, nil
}

// Close closes the database connection
func (mt *MessageTracker) Close() error {
	return mt.db.Close()
}
