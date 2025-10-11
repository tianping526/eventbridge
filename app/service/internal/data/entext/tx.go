package entext

import (
	"context"

	"github.com/tianping526/eventbridge/app/service/internal/data/ent"
)

// WithTx runs callbacks in a transaction.
func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) (err error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			err = tx.Rollback()
			if err != nil {
				return
			}
			panic(v)
		}
	}()
	if err = fn(tx); err != nil {
		if rer := tx.Rollback(); rer != nil {
			return rer
		}
		return err
	}
	err = tx.Commit()
	return err
}
