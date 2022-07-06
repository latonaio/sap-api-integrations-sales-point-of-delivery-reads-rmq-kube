package sap_api_caller

import (
	"fmt"
	"io/ioutil"
	"net/http"
	sap_api_output_formatter "sap-api-integrations-sales-point-of-delivery-reads-rmq-kube/SAP_API_Output_Formatter"
	"strings"
	"sync"

	"github.com/latonaio/golang-logging-library-for-sap/logger"
	"golang.org/x/xerrors"
)

type RMQOutputter interface {
	Send(sendQueue string, payload map[string]interface{}) error
}

type SAPAPICaller struct {
	baseURL      string
	apiKey       string
	outputQueues []string
	outputter    RMQOutputter
	log          *logger.Logger
}

func NewSAPAPICaller(baseUrl string, outputQueueTo []string, outputter RMQOutputter, l *logger.Logger) *SAPAPICaller {
	return &SAPAPICaller{
		baseURL:      baseUrl,
		apiKey:       GetApiKey(),
		outputQueues: outputQueueTo,
		outputter:    outputter,
		log:          l,
	}
}

func (c *SAPAPICaller) AsyncGetSalesPointOfDelivery(salesPointOfDeliveryID string, accepter []string) {
	wg := &sync.WaitGroup{}
	wg.Add(len(accepter))
	for _, fn := range accepter {
		switch fn {
		case "SalesPointOfDeliveryCollection":
			func() {
				c.SalesPointOfDeliveryCollection(salesPointOfDeliveryID)
				wg.Done()
			}()
		default:
			wg.Done()
		}
	}

	wg.Wait()
}

func (c *SAPAPICaller) SalesPointOfDeliveryCollection(salesPointOfDeliveryID string) {
	SalesPointOfDeliveryCollectionData, err := c.callSalesPointOfDeliverySrvAPIRequirementSalesPointOfDeliveryCollection("UtilitiesSalesPointOfDeliveryCollection", salesPointOfDeliveryID)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": SalesPointOfDeliveryCollectionData, "function": "SalesPointOfDeliveryCollection"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(SalesPointOfDeliveryCollectionData)
}

func (c *SAPAPICaller) callSalesPointOfDeliverySrvAPIRequirementSalesPointOfDeliveryCollection(api, salesPointOfDeliveryID string) ([]sap_api_output_formatter.SalesPointOfDeliveryCollection, error) {
	url := strings.Join([]string{c.baseURL, "c4codataapi", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithSalesPointOfDeliveryCollection(req, salesPointOfDeliveryID)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToSalesPointOfDeliveryCollection(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) setHeaderAPIKeyAccept(req *http.Request) {
	req.Header.Set("APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
}

func (c *SAPAPICaller) getQueryWithSalesPointOfDeliveryCollection(req *http.Request, salesPointOfDeliveryID string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("SalesPointOfDeliveryID eq '%s'", salesPointOfDeliveryID))
	req.URL.RawQuery = params.Encode()
}
