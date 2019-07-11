package dataloaders

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yasnov/goreceipt/api"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type ctxKeyType struct{ name string }

var ctxKey = ctxKeyType{"userCtx"}

type Loaders struct {
	ItemsByReceipt *ItemSliceLoader
}

func LoaderMiddleware(db *sqlx.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ldrs := Loaders{}

		// set this to zero what happens without dataloading
		wait := 250 * time.Microsecond

		ldrs.ItemsByReceipt = &ItemSliceLoader{
			wait:     wait,
			maxBatch: 200,
			fetch: func(ids []int) ([][]*api.Item, []error) {
				placeholders := make([]string, len(ids))
				args := make([]interface{}, len(ids))
				for i := 0; i < len(ids); i++ {
					placeholders[i] = "$" + strconv.Itoa(i+1)
					args[i] = ids[i]
				}

				sqlQuery := "SELECT * from item WHERE receipt_id IN (" +
					strings.Join(placeholders, ",") + ")"
				rows, err := db.Query(sqlQuery,
					args...,
				)
				if err != nil {
					panic(err)
				}
				items := make(map[int][]*api.Item, len(ids))
				errors := make([]error, len(ids))
				for rows.Next() {
					var itemID, receiptID int
					var text sql.NullString
					var price, amount decimal.NullDecimal
					err = rows.Scan(&itemID, &text, &price, &amount, &receiptID)
					item := &api.Item{
						ID:     itemID,
						Amount: amount,
						Price:  price,
						Text:   text.String,
					}
					if err != nil {
						errors[item.ID] = err
					}
					items[receiptID] = append(items[receiptID], item)
				}

				output := make([][]*api.Item, len(ids))
				for i, id := range ids {
					output[i] = items[id]
				}
				return output, errors
			},
		}

		dlCtx := context.WithValue(r.Context(), ctxKey, ldrs)
		next.ServeHTTP(w, r.WithContext(dlCtx))
	})
}

func CtxLoaders(ctx context.Context) Loaders {
	return ctx.Value(ctxKey).(Loaders)
}
