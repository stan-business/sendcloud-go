package servicepoint

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/afosto/sendcloud-go"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	NoSuitableServicePointFound = errors.New("could not find a matching service point in SendCloud")
)

type Client struct {
	apiKey    string
	apiSecret string
}

type Matcher struct {
	Country     string
	SPID        *string
	Carrier     *string
	City        *string
	PostalCode  *string
	HouseNumber *string
	Latitude    *float64
	Longitude   *float64
	Address     *string
	Weight      *float64
	Radius      *int64
}

func New(apiKey string, apiSecret string) *Client {
	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

func (service Client) GetServicePoints(servicePoint Matcher) (sendcloud.ServicePointList, error) {
	if len(servicePoint.Country) == 0 {
		return nil, errors.New("country is required")
	}

	//prepare bounding box url
	uri, _ := url.Parse("https://servicePoints.sendcloud.sc/api/v2/service-points/")
	params := map[string]string{
		"country":      strings.ToUpper(servicePoint.Country),
		"access_token": service.apiKey,
	}
	if servicePoint.Address != nil {
		params["address"] = *servicePoint.Address
	}
	if servicePoint.City != nil {
		params["city"] = *servicePoint.City
	}
	if servicePoint.HouseNumber != nil {
		params["house_number"] = *servicePoint.HouseNumber
	}
	if servicePoint.Weight != nil {
		params["weight"] = fmt.Sprintf("%.4f", *servicePoint.Weight)
	}
	if servicePoint.Carrier != nil {
		params["carrier"] = *servicePoint.Carrier
	}
	if servicePoint.Radius != nil {
		params["radius"] = fmt.Sprintf("%d", *servicePoint.Radius)
	}

	paramsContainer := uri.Query()
	for key, value := range params {
		paramsContainer.Add(key, value)
	}
	uri.RawQuery = paramsContainer.Encode()

	servicePoints := sendcloud.ServicePointList{}
	if err := sendcloud.Request("GET", uri.String(), nil, service.apiKey, service.apiSecret, &servicePoints); err != nil {
		return nil, err
	}

	return servicePoints, nil
}

// Returns the sendcloud pickup point ID mapped from a SPID ID
func (service Client) GetServicePoint(servicePoint Matcher) (int, error) {
	//prepare bounding box url
	uri, _ := url.Parse("https://servicePoints.sendcloud.sc/api/v2/service-points/")
	params := map[string]string{
		"country":      strings.ToUpper(servicePoint.Country),
		"ne_latitude":  fmt.Sprintf("%.4f", *servicePoint.Latitude+0.06),
		"sw_latitude":  fmt.Sprintf("%.4f", *servicePoint.Latitude-0.06),
		"ne_longitude": fmt.Sprintf("%.4f", *servicePoint.Longitude+0.06),
		"sw_longitude": fmt.Sprintf("%.4f", *servicePoint.Longitude-0.06),
		"access_token": service.apiKey,
		"carrier":      *servicePoint.Carrier,
	}
	paramsContainer := uri.Query()
	for key, value := range params {
		paramsContainer.Add(key, value)
	}
	uri.RawQuery = paramsContainer.Encode()

	servicePoints := sendcloud.ServicePointList{}
	if err := sendcloud.Request("GET", uri.String(), nil, service.apiKey, service.apiSecret, &servicePoints); err != nil {
		return 0, err
	}

	matching := unaccent(fmt.Sprintf("%s %s", *servicePoint.PostalCode, *servicePoint.HouseNumber))

	for _, sp := range servicePoints {

		if unaccent(sp.Identifier()) == matching {
			return sp.ID, nil
		}
		if sp.Code == *servicePoint.SPID {
			return sp.ID, nil
		}
	}

	return 0, NoSuitableServicePointFound
}

func unaccent(string string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, string)
	return result
}
