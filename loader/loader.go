package loader

import (
	"database/sql"
	"time"

	"github.com/yasnov/goreceipt/api"
	"github.com/yasnov/goreceipt/api/resolver"
	"github.com/yasnov/goreceipt/loader/providers"
	"github.com/yasnov/goreceipt/logger"

	"github.com/jmoiron/sqlx"
)

func StartLoader(res *resolver.Resolver) {

	type Provider interface {
		NewReceipt(*api.Receipt)
		Parse() error
		GetItems() []*api.Item
	}
	listOfProviders := []Provider{&providers.NalogProvider{}, &providers.OFD1Provider{}, &providers.TaxcomProvider{}}
	log := logger.CreateNewProvidersLogger()
	for {
		select {
		case receipt := <-res.UndefinedReceiptsChannel:
			var errorStrings []string
			var itemsList []*api.Item
			var successProviderTime time.Duration
			startTime := time.Now()
			for _, pr := range listOfProviders {
				provider_time_start := time.Now()
				pr.NewReceipt(receipt)
				err := pr.Parse()
				if err != nil {
					errorStrings = append(errorStrings, err.Error())
				}
				if receipt.Provider != "" {
					successProviderTime = time.Now().Sub(provider_time_start)
					itemsList = pr.GetItems()
					break
				}
			}
			allTime := time.Now().Sub(startTime)
			if itemsList != nil {
				receipt.Status = "LOADED"
			} else {
				receipt.Status = "FAILED"
			}
			log.WithFields(map[string]interface{}{
				"receiptID":         receipt.ID,
				"status":            receipt.Status,
				"provider":          receipt.Provider,
				"provider_time_sec": successProviderTime.Seconds(),
				"all_time_sec":      allTime.Seconds(),
				"errors":            errorStrings,
			}).Info("Receipt loaded")

			if _, found := res.UpdateReceiptChannel[receipt.User]; found {
				res.UpdateReceiptChannel[receipt.User] <- receipt
			}
			err := SaveItems(res.DB, receipt.ID, itemsList)
			if err != nil {
				errorStrings = append(errorStrings, err.Error())
			}
			err = UpdateReceiptAfterParsing(res.DB, receipt)
			if err != nil {
				errorStrings = append(errorStrings, err.Error())
			}
		}
	}
}

func UpdateReceiptAfterParsing(db *sqlx.DB, rec *api.Receipt) error {
	placeID, err := SavePlace(db, rec)
	if err != nil {
		return err
	}
	sqlStatement := `
		UPDATE receipt
		SET date = $2, provider = $3, status = $4, place_id = $5, fn = $6, fp = $7, fd = $8
		WHERE id = $1;`
	_, err = db.Exec(sqlStatement, rec.ID, rec.Date, rec.Provider, rec.Status,
		placeID, rec.Fn, rec.Fd, rec.Fp)
	if err != nil {
		return err
	}
	return nil
}

func SavePlace(db *sqlx.DB, rec *api.Receipt) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM place WHERE text = $1;", rec.Place).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return -1, err
	}
	if id != 0 {
		return id, nil
	} else {
		err = db.QueryRow("INSERT INTO place (text) VALUES ($1) RETURNING id;", rec.Place).Scan(&id)
		if err != nil {
			return -1, err
		}
		return id, nil
	}
}

func SaveItems(db *sqlx.DB, recID int, items []*api.Item) error {
	sqlStatement := `
		INSERT INTO item (text, price, amount, receipt_id)
		VALUES ($1, $2, $3, $4);`
	for _, i := range items {
		_, err := db.Exec(sqlStatement, i.Text, i.Price, i.Amount, recID)
		if err != nil {
			return err
		}
	}
	return nil
}
