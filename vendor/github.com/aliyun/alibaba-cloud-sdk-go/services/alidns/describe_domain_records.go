package alidns

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// DescribeDomainRecords invokes the alidns.DescribeDomainRecords API synchronously
// api document: https://help.aliyun.com/api/alidns/describedomainrecords.html
func (client *Client) DescribeDomainRecords(request *DescribeDomainRecordsRequest) (response *DescribeDomainRecordsResponse, err error) {
	response = CreateDescribeDomainRecordsResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeDomainRecordsWithChan invokes the alidns.DescribeDomainRecords API asynchronously
// api document: https://help.aliyun.com/api/alidns/describedomainrecords.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeDomainRecordsWithChan(request *DescribeDomainRecordsRequest) (<-chan *DescribeDomainRecordsResponse, <-chan error) {
	responseChan := make(chan *DescribeDomainRecordsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeDomainRecords(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// DescribeDomainRecordsWithCallback invokes the alidns.DescribeDomainRecords API asynchronously
// api document: https://help.aliyun.com/api/alidns/describedomainrecords.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) DescribeDomainRecordsWithCallback(request *DescribeDomainRecordsRequest, callback func(response *DescribeDomainRecordsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeDomainRecordsResponse
		var err error
		defer close(result)
		response, err = client.DescribeDomainRecords(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// DescribeDomainRecordsRequest is the request struct for api DescribeDomainRecords
type DescribeDomainRecordsRequest struct {
	*requests.RpcRequest
	ValueKeyWord string           `position:"Query" name:"ValueKeyWord"`
	Line         string           `position:"Query" name:"Line"`
	GroupId      requests.Integer `position:"Query" name:"GroupId"`
	DomainName   string           `position:"Query" name:"DomainName"`
	OrderBy      string           `position:"Query" name:"OrderBy"`
	Type         string           `position:"Query" name:"Type"`
	PageNumber   requests.Integer `position:"Query" name:"PageNumber"`
	UserClientIp string           `position:"Query" name:"UserClientIp"`
	PageSize     requests.Integer `position:"Query" name:"PageSize"`
	SearchMode   string           `position:"Query" name:"SearchMode"`
	Lang         string           `position:"Query" name:"Lang"`
	KeyWord      string           `position:"Query" name:"KeyWord"`
	TypeKeyWord  string           `position:"Query" name:"TypeKeyWord"`
	RRKeyWord    string           `position:"Query" name:"RRKeyWord"`
	Direction    string           `position:"Query" name:"Direction"`
	Status       string           `position:"Query" name:"Status"`
}

// DescribeDomainRecordsResponse is the response struct for api DescribeDomainRecords
type DescribeDomainRecordsResponse struct {
	*responses.BaseResponse
	RequestId     string                               `json:"RequestId" xml:"RequestId"`
	TotalCount    int64                                `json:"TotalCount" xml:"TotalCount"`
	PageNumber    int64                                `json:"PageNumber" xml:"PageNumber"`
	PageSize      int64                                `json:"PageSize" xml:"PageSize"`
	DomainRecords DomainRecordsInDescribeDomainRecords `json:"DomainRecords" xml:"DomainRecords"`
}

// CreateDescribeDomainRecordsRequest creates a request to invoke DescribeDomainRecords API
func CreateDescribeDomainRecordsRequest() (request *DescribeDomainRecordsRequest) {
	request = &DescribeDomainRecordsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Alidns", "2015-01-09", "DescribeDomainRecords", "Alidns", "openAPI")
	return
}

// CreateDescribeDomainRecordsResponse creates a response to parse from DescribeDomainRecords response
func CreateDescribeDomainRecordsResponse() (response *DescribeDomainRecordsResponse) {
	response = &DescribeDomainRecordsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
