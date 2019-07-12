package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/yasnov/goreceipt/api"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	firstURL        = "https://consumer.1-ofd.ru/#/landing"
	secondURL       = "https://consumer.1-ofd.ru/api/messages"
	findURL         = "https://consumer.1-ofd.ru/api/tickets/find-ticket"
	getOFDResultURL = "https://consumer.1-ofd.ru/api/tickets/ticket/%s"
)

type OFD1Provider struct {
	client *http.Client
	Items  []*api.Item
	*api.Receipt
}

func (r *OFD1Provider) NewReceipt(receipt *api.Receipt) {
	r.Receipt = receipt
	r.Items = []*api.Item{}
	r.client = newWebClient()
}

func (r *OFD1Provider) GetItems() []*api.Item {
	return r.Items
}

func (r *OFD1Provider) Parse() error {
	err := r.first()
	if err != nil {
		return errors.Wrap(err, "1-OFD")
	}
	token, err := r.second()
	if err != nil {
		return errors.Wrap(err, "1-OFD")
	}
	uid, err := r.find(token)
	if err != nil {
		return errors.Wrap(err, "1-OFD")
	}
	body, err := r.get(token, uid)
	if err != nil {
		return errors.Wrap(err, "1-OFD")
	}
	r.Place, err = jsonparser.GetString(body, "orgTitle")
	if err != nil {
		return errors.Wrap(err, "1-OFD")
	}
	_, err = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		item := api.Item{}
		item.Text, _ = jsonparser.GetString(value, "commodity", "name")
		amountFloat, err := jsonparser.GetFloat(value, "commodity", "quantity")
		if err == nil {
			item.Amount = decimal.NullDecimal{Decimal: decimal.NewFromFloat(amountFloat), Valid: true}
		}
		priceFloat, err := jsonparser.GetFloat(value, "commodity", "sum")
		if err == nil {
			item.Price = decimal.NullDecimal{Decimal: decimal.NewFromFloat(priceFloat), Valid: true}
		}
		r.Items = append(r.Items, &item)
	}, "ticket", "items")
	if err != nil {
		return errors.Wrap(err, "1-OFD")
	}
	r.Provider = "1-OFD"
	return nil
}

func setOfdHeaders(r *http.Request) {
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 "+
		" (KHTML, like Gecko) Chrome/65.0.3325.181 Safari/537.36")
}

func (r *OFD1Provider) first() error {
	req, err := http.NewRequest(http.MethodGet, firstURL, nil)
	if err != nil {
		return err
	}
	setOfdHeaders(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s", resp.Status)
	}
	return nil
}

func (r *OFD1Provider) second() (*string, error) {
	req, err := http.NewRequest(http.MethodGet, secondURL, nil)
	if err != nil {
		return nil, err
	}
	setOfdHeaders(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "XSRF-TOKEN" {
			return &cookie.Value, nil
		}
	}
	return nil, nil
}

func (r *OFD1Provider) find(token *string) (*string, error) {
	values := map[string]string{"fiscalDocumentNumber": r.Fd, "fiscalDriveId": r.Fn, "fiscalId": r.Fp}
	jsonValue, _ := json.Marshal(values)
	req, err := http.NewRequest(http.MethodPost, findURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	setOfdHeaders(req)
	req.Header.Add("X-XSRF-TOKEN", *token)
	req.AddCookie(&http.Cookie{Name: "PLAY_LANG", Value: "ru"})
	req.AddCookie(&http.Cookie{Name: "X-XSRF-TOKEN", Value: *token})
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	uid, err := jsonparser.GetString(body, "uid")
	if err != nil {
		return nil, err
	}
	return &uid, nil
}

func (r *OFD1Provider) get(token, uid *string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(getOFDResultURL, *uid), nil)
	if err != nil {
		return nil, err
	}
	setOfdHeaders(req)
	req.Header.Add("X-XSRF-TOKEN", *token)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return body, nil
}
