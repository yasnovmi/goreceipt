package api

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/shopspring/decimal"
)

type Item struct {
	ID     int                 `json:"id"`
	Text   string              `json:"name"`
	Price  decimal.NullDecimal `json:"price"`
	Amount decimal.NullDecimal `json:"amount"`
}

type NewReceipt struct {
	Fn   string              `json:"fn"`
	Fd   string              `json:"fd"`
	Fp   string              `json:"fp"`
	Date string              `json:"date"`
	Sum  decimal.NullDecimal `json:"sum"`
}

type Receipt struct {
	ID       int                 `json:"id"`
	Fn       string              `json:"fn"`
	Fd       string              `json:"fd"`
	Fp       string              `json:"fp"`
	Date     time.Time           `json:"date"`
	Sum      decimal.NullDecimal `json:"sum"`
	Place    string              `json:"place"`
	Provider string              `json:"provider"`
	Status   string              `json:"status"`
	//Items    []*Item   `json:"items"`
	User int
}

type ReceiptFilters struct {
	DateFrom   *time.Time           `json:"date_from"`
	DateTo     *time.Time           `json:"date_to"`
	SummaryMin *decimal.NullDecimal `json:"summary_min"`
	SummaryMax *decimal.NullDecimal `json:"summary_max"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthPayload struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type LogoutResult struct {
	User User `json:"user"`
}

type SignUpInput struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type User struct {
	ID       int    `json:"ID"`
	Username string `json:"username"`
}

func MarshalID(id int) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.Quote(fmt.Sprintf("%d", id)))
	})
}

func UnmarshalID(v interface{}) (int, error) {
	id, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("ids must be strings")
	}
	i, e := strconv.Atoi(id)
	return int(i), e
}

func MarshalDecimal(d decimal.NullDecimal) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		if d.Valid {
			_, err := io.WriteString(w, strconv.Quote(d.Decimal.String()))
			if err != nil {
				log.Print(strconv.Quote(d.Decimal.String()))
			}
		} else {
			_, err := io.WriteString(w, strconv.Quote(""))
			if err != nil {
				log.Print(strconv.Quote(d.Decimal.String()))
			}
		}
	})
}

func UnmarshalDecimal(v interface{}) (decimal.NullDecimal, error) {
	var dec decimal.NullDecimal
	err := dec.Scan(v)
	return dec, err
}
