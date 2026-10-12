package services

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

const (
	// lockRetryAttempts is how many times a transaction is run in total when
	// MySQL keeps picking it as a deadlock victim.
	lockRetryAttempts = 4
	lockRetryBackoff  = 15 * time.Millisecond

	mysqlLockWaitTimeout = 1205
	mysqlDeadlock        = 1213
)

// isLockConflict reports whether err is MySQL rolling a transaction back
// because of a deadlock or a lock wait timeout. Both mean "run it again".
func isLockConflict(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	return mysqlErr.Number == mysqlDeadlock || mysqlErr.Number == mysqlLockWaitTimeout
}

// retryOnLockConflict runs fn, a whole transaction, again when it failed
// with a lock conflict, up to lockRetryAttempts times with a short, jittered
// pause. Any other error, and the last conflict, is returned as is.
func retryOnLockConflict(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 1; attempt <= lockRetryAttempts; attempt++ {
		if err = fn(); err == nil || !isLockConflict(err) {
			return err
		}
		if attempt == lockRetryAttempts {
			break
		}
		pause := time.Duration(attempt)*lockRetryBackoff + rand.N(lockRetryBackoff)
		select {
		case <-ctx.Done():
			return err
		case <-time.After(pause):
		}
	}
	return err
}
