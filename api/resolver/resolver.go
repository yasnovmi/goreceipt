package resolver

import (
	"github.com/yasnov/goreceipt"
	"github.com/yasnov/goreceipt/api"
	"github.com/yasnov/goreceipt/api/dataloaders"
	cnf "github.com/yasnov/goreceipt/config"
	"github.com/yasnov/goreceipt/tools"

	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/jackc/pgx/pgtype"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct {
	DB                       *sqlx.DB
	mu                       sync.RWMutex
	UndefinedReceiptsChannel chan *api.Receipt
	UpdateReceiptChannel     map[int]chan *api.Receipt
}

func NewResolver(db *sqlx.DB) *Resolver {
	return &Resolver{DB: db,
		UpdateReceiptChannel:     make(map[int]chan *api.Receipt, 5),
		UndefinedReceiptsChannel: make(chan *api.Receipt, 10)}
}
func NewRootResolvers(res *Resolver) goreceipt.Config {
	return goreceipt.Config{
		Resolvers: res,
	}
}
func (r *Resolver) Mutation() goreceipt.MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() goreceipt.QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Receipt() goreceipt.ReceiptResolver {
	return &receiptResolver{r}
}
func (r *Resolver) Subscription() goreceipt.SubscriptionResolver {
	return &subscriptionResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) Signup(ctx context.Context, params api.SignUpInput) (*api.AuthPayload, error) {
	panic("implement me")
}

func (r *mutationResolver) Login(ctx context.Context, params api.LoginInput) (*api.AuthPayload, error) {
	panic("implement me")
}

func (r *mutationResolver) Logout(ctx context.Context) (*api.LogoutResult, error) {
	panic("implement me")
}

func (r *mutationResolver) CreateReceiptByQr(ctx context.Context, code string) (int, error) {
	v, err := url.ParseQuery(code)
	if err != nil {
		return -1, err
	}
	date, err := tools.DateFromString(v.Get("t"))
	if err != nil {
		graphql.AddError(ctx, fmt.Errorf("Date in wrong format"))
	}
	newReceipt := &api.Receipt{
		User: cnf.Config.UserTestID,
		Fd:   v.Get("i"),
		Fp:   v.Get("fp"),
		Fn:   v.Get("fn"),
		Date: date,
	}
	err = newReceipt.Sum.Scan(v.Get("s"))
	if err != nil {
		graphql.AddError(ctx, fmt.Errorf("Summary in wrong format"))
	}
	if newReceipt.Fp == "" || newReceipt.Fd == "" || newReceipt.Fn == "" || !newReceipt.Sum.Valid || newReceipt.Date.IsZero() {
		return -1, fmt.Errorf("Not enough arguments")
	}
	var id int
	id, err = CheckReceiptExist(r.DB, newReceipt)
	if err != nil {
		return -1, err
	}
	if id <= 0 {
		insertReceipt := `
						INSERT INTO receipt (fn, fd, fp, status, date, sum)
						VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;`
		err = r.DB.QueryRow(insertReceipt, newReceipt.Fn, newReceipt.Fd,
			newReceipt.Fp, "UNDEFINED", date, newReceipt.Sum).Scan(&id)
		if err != nil {
			return -1, err
		}
		newReceipt.ID = id
		r.UndefinedReceiptsChannel <- newReceipt
	} else {
		graphql.AddErrorf(ctx, "Receipt already exists")
	}
	return id, nil
}

func (r *mutationResolver) CreateReceipt(ctx context.Context, input api.NewReceipt) (int, error) {
	dateFromString, err := tools.DateFromString(input.Date)
	if err != nil {
		return -1, err
	}
	var id int
	newReceipt := &api.Receipt{
		User: 1,
		Fd:   input.Fd,
		Fp:   input.Fp,
		Fn:   input.Fn,
		Sum:  input.Sum,
		Date: dateFromString,
	}
	id, err = CheckReceiptExist(r.DB, newReceipt)
	if err != nil {
		return -1, err
	}
	if id == 0 {
		insertReceipt := `
						INSERT INTO receipt (fn, fd, fp, status, date, sum)
						VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;`
		err = r.DB.QueryRow(insertReceipt, input.Fn, input.Fd, input.Fp, "UNDEFINED", dateFromString, input.Sum).Scan(&id)
		if err != nil {
			return -1, err
		}
		newReceipt.ID = id
		r.UndefinedReceiptsChannel <- newReceipt
	} else {
		graphql.AddErrorf(ctx, "Receipt already exists")
	}
	return id, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Receipts(ctx context.Context, input api.ReceiptFilters) ([]*api.Receipt, error) {
	sqlStatement := `
			SELECT r.id, r.sum, r.fn, r.fp, r.fd, date, r.provider, r.status, p.text
			FROM receipt r
			LEFT JOIN place p on r.place_id = p.id
			WHERE ($1::double precision IS NULL OR (sum IS NOT NULL AND sum < $1::double precision))
			AND ($2::double precision IS NULL OR (sum IS NOT NULL AND sum > $2::double precision))
			AND ($3::timestamp IS NULL OR (date IS NOT NULL AND date < $3::timestamp))
			AND ($4::timestamp IS NULL OR (date IS NOT NULL AND date > $4::timestamp))
			order by -r.id`
	rows, err := r.DB.Queryx(sqlStatement, input.SummaryMax, input.SummaryMin, input.DateTo, input.DateFrom)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var receipts []*api.Receipt
	for rows.Next() {
		var fn, fp, fd, provider, text, status sql.NullString
		var id int
		var sum decimal.NullDecimal
		var date pgtype.Timestamptz
		err = rows.Scan(&id, &sum, &fn, &fp, &fd, &date, &provider, &status, &text)
		if err != nil {
			return nil, err
		}
		rcpt := api.Receipt{
			ID:       id,
			Fp:       fp.String,
			Fn:       fn.String,
			Fd:       fd.String,
			Sum:      sum,
			Status:   status.String,
			Date:     date.Time,
			Place:    text.String,
			Provider: provider.String,
		}
		receipts = append(receipts, &rcpt)
	}
	return receipts, nil
}

func (r *queryResolver) Receipt(ctx context.Context, receiptID int) (*api.Receipt, error) {
	var fn, fp, fd, provider, text sql.NullString
	var sum decimal.NullDecimal
	var date pgtype.Timestamp
	sqlStatement := `
		SELECT sum, fn, fp, fd, date, provider, p.text
		FROM receipt r
		LEFT JOIN place p on r.place_id = p.id
		WHERE r.id = $1;`
	err := r.DB.QueryRowx(sqlStatement, receiptID).Scan(&sum, &fn, &fp, &fd, &date, &provider, &text)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ReceiptID is not correct")
	}
	if err != nil {
		return nil, err
	}
	receipt := &api.Receipt{
		ID:       receiptID,
		Fn:       fn.String,
		Fp:       fp.String,
		Fd:       fd.String,
		Place:    text.String,
		Date:     date.Time,
		Sum:      sum,
		Provider: provider.String,
	}

	return receipt, nil
}

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) ReceiptUpdate(ctx context.Context, userID int) (<-chan *api.Receipt, error) {
	receiptEvent := make(chan *api.Receipt, 1)
	go func() {
		<-ctx.Done()
		r.mu.Lock()
		delete(r.UpdateReceiptChannel, userID)
		r.mu.Unlock()
	}()
	r.mu.Lock()
	r.UpdateReceiptChannel[userID] = receiptEvent
	r.mu.Unlock()
	return receiptEvent, nil
}

type receiptResolver struct{ *Resolver }

func (r *receiptResolver) Items(ctx context.Context, obj *api.Receipt) ([]*api.Item, error) {
	return dataloaders.CtxLoaders(ctx).ItemsByReceipt.Load(obj.ID)
}

func CheckReceiptExist(db *sqlx.DB, rec *api.Receipt) (int, error) {
	var id int
	err := db.QueryRowx("SELECT id FROM receipt WHERE fd = $1 AND fp = $2 AND fn = $3;",
		rec.Fd, rec.Fp, rec.Fn).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if id != 0 {
		return id, nil
	}
	return 0, nil
}
