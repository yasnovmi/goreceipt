package providers

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/yasnov/goreceipt/api"
	. "github.com/yasnov/goreceipt/config"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	loginURL   = "https://proverkacheka.nalog.ru:9999/v1/mobile/users/login"
	searchURL  = "https://proverkacheka.nalog.ru:9999/v1/ofds/*/inns/*/fss/%s/operations/1/tickets/%s?fiscalSign=%s&date=%s&sum=%d"
	getDataURL = "https://proverkacheka.nalog.ru:9999/v1/inns/*/kkts/*/fss/%s/tickets/%s?fiscalSign=%s&sendToEmail=no"
)

type NalogProvider struct {
	client *http.Client
	Items  []*api.Item
	*api.Receipt
	*ListOfURLs
}

type ListOfURLs struct {
	loginURL   string
	searchURL  string
	getDataURL string
}

func (r *NalogProvider) NewReceipt(receipt *api.Receipt) {
	r.Receipt = receipt
	r.Items = []*api.Item{}
	r.client = newWebClient()
}

func SetNalogHeaders(r *http.Request) {
	r.SetBasicAuth(Config.Nalog.Login, Config.Nalog.Password)
	r.Header.Add("Device-Id", "ANDROID_ID")
	r.Header.Add("Device-OS", "Adnroid 6.0.1")
	r.Header.Add("Version", "2")
	r.Header.Add("ClientVersion", "1.4.4.4")
}

func (r *NalogProvider) Parse() error {
	err := r.login()
	if err != nil {
		return errors.Wrap(err, "Nalog")
	}
	err = r.search()
	if err != nil {
		return errors.Wrap(err, "Nalog")
	}
	body, err := r.getData(0)
	if err != nil {
		return errors.Wrap(err, "Nalog")
	}
	r.Place, err = jsonparser.GetString(body, "document", "receipt", "user")
	if err != nil || r.Place == "" {
		r.Place, _ = jsonparser.GetString(body, "document", "receipt", "userInn")
	}
	_, err = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		item := api.Item{}
		item.Text, _ = jsonparser.GetString(value, "name")
		amountFloat, err := jsonparser.GetFloat(value, "quantity")
		if err == nil {
			item.Amount = decimal.NullDecimal{Decimal: decimal.NewFromFloat(amountFloat), Valid: true}
		}
		priceFloat, err := jsonparser.GetFloat(value, "price")
		if err == nil {
			item.Price = decimal.NullDecimal{Decimal: decimal.NewFromFloat(priceFloat / 100), Valid: true}
		}
		r.Items = append(r.Items, &item)
	}, "document", "receipt", "items")
	if err != nil {
		return errors.Wrap(err, "Nalog")
	}
	r.Provider = "NALOG"
	return nil
}

func (r *NalogProvider) login() error {
	req, err := http.NewRequest(http.MethodGet, loginURL, nil)
	if err != nil {
		return err
	}
	SetNalogHeaders(req)
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

func (r *NalogProvider) search() error {
	sumFloat, _ := r.Sum.Decimal.Float64()
	url := fmt.Sprintf(searchURL, r.Fn, r.Fd, r.Fp, r.Date.Format("20060102T1504"), int(sumFloat*100))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	SetNalogHeaders(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("%s", resp.Status)
	}

	return nil
}

func (r *NalogProvider) getData(try int) ([]byte, error) {
	url := fmt.Sprintf(getDataURL, r.Fn, r.Fd, r.Fp)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	SetNalogHeaders(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 202 && try <= 8 {
		time.Sleep(time.Millisecond * 300)
		return r.getData(try + 1)
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return body, nil
}

func (r *NalogProvider) GetItems() []*api.Item {
	return r.Items
}
