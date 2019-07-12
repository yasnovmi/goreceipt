package providers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yasnov/goreceipt/api"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const taxcomGetRcptURL = "https://receipt.taxcom.ru/v01/show?fp=%s&s=%s"

type TaxcomProvider struct {
	client *http.Client
	Items  []*api.Item
	*api.Receipt
}

func (r *TaxcomProvider) NewReceipt(receipt *api.Receipt) {
	r.Receipt = receipt
	r.Items = []*api.Item{}
	r.client = newWebClient()
}

func (r *TaxcomProvider) GetItems() []*api.Item {
	return r.Items
}

func (r *TaxcomProvider) Parse() error {
	body, err := r.search()
	if err != nil {
		return errors.Wrap(err, "Taxcom")
	}
	defer body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return errors.Wrap(err, "Taxcom")
	}
	isFind := doc.Find("jumbotron notfound")
	if isFind != nil {
		return errors.Wrap(fmt.Errorf("Receipt not found"), "Taxcom")
	}

	doc.Find("span").Each(func(i int, s *goquery.Selection) {
		if r.Fd == "" && s.HasClass("value receipt-value-1040") {
			r.Fd = strings.TrimSpace(s.Text())
		}
		if r.Fn == "" && s.HasClass("value receipt-value-1041") {
			r.Fn = strings.TrimSpace(s.Text())
		}
		if r.Date.IsZero() && s.HasClass("value receipt-value-1012") {
			fmt.Println(s.Text())
			r.Date, _ = time.Parse("02.01.2006 15:04", strings.TrimSpace(s.Text()))
		}
	})
	r.Place = strings.TrimSpace(doc.Find(".receipt-subtitle").Text())

	doc.Find(".item").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Find(".receipt-row-1").First().Text())
		tabletCol := s.Find(".receipt-row-2").First().Find(".receipt-col1").First().Find("span")
		price := strings.TrimSpace(tabletCol.Last().Text())
		amount := strings.TrimSpace(tabletCol.First().Text())

		fmt.Println(text, price, amount)
		var item api.Item
		priceDec, err := decimal.NewFromString(price)
		if err == nil {
			item.Price = decimal.NullDecimal{Decimal: priceDec, Valid: true}
		}
		amountDec, err := decimal.NewFromString(amount)
		if err == nil {
			item.Amount = decimal.NullDecimal{Decimal: amountDec, Valid: true}
		}
		item.Text = text
		r.Items = append(r.Items, &item)
	})
	r.Provider = "TAXCOM"
	return nil
}

func (r *TaxcomProvider) search() (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(taxcomGetRcptURL, r.Fp, r.Sum.Decimal.String()), nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return resp.Body, nil
}
