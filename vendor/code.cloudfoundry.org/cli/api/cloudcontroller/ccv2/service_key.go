package ccv2

import (
	"bytes"
	"encoding/json"
	"net/url"

	"code.cloudfoundry.org/cli/api/cloudcontroller"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/internal"
)

// ServiceKey represents a Cloud Controller Service Key.
type ServiceKey struct {
	// GUID is the unique Service Key identifier.
	GUID string

	// Name is the name of the service binding
	Name string

	// ServiceInstanceGUID is the associated service GUID.
	ServiceInstanceGUID string

	Credentials map[string]interface{}
}

// UnmarshalJSON helps unmarshal a Cloud Controller Service Key response.
func (serviceKey *ServiceKey) UnmarshalJSON(data []byte) error {
	var ccServiceKey struct {
		Metadata internal.Metadata
		Entity   struct {
			ServiceInstanceGUID string                 `json:"service_instance_guid"`
			Name                string                 `json:"name"`
			Credentials         map[string]interface{} `json:"credentials"`
		} `json:"entity"`
	}
	err := cloudcontroller.DecodeJSON(data, &ccServiceKey)
	if err != nil {
		return err
	}
	serviceKey.GUID = ccServiceKey.Metadata.GUID
	serviceKey.ServiceInstanceGUID = ccServiceKey.Entity.ServiceInstanceGUID
	serviceKey.Name = ccServiceKey.Entity.Name
	serviceKey.Credentials = ccServiceKey.Entity.Credentials
	return nil
}

// serviceKeyRequestBody represents the body of the service binding create
// request.
type serviceKeyRequestBody struct {
	ServiceInstanceGUID string                 `json:"service_instance_guid"`
	Name                string                 `json:"name"`
	Parameters          map[string]interface{} `json:"parameters"`
}

func (client *Client) CreateServiceKey(serviceInstanceGUID string, keyName string, acceptsIncomplete bool, parameters map[string]interface{}) (ServiceKey, Warnings, error) {
	requestBody := serviceKeyRequestBody{
		ServiceInstanceGUID: serviceInstanceGUID,
		Name:                keyName,
		Parameters:          parameters,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return ServiceKey{}, nil, err
	}

	request, err := client.newHTTPRequest(requestOptions{
		RequestName: internal.PostServiceKeyRequest,
		Body:        bytes.NewReader(bodyBytes),
		Query:       url.Values{"accepts_incomplete": {"false"}},
	})
	if err != nil {
		return ServiceKey{}, nil, err
	}

	var serviceKey ServiceKey
	response := cloudcontroller.Response{
		Result: &serviceKey,
	}

	err = client.connection.Make(request, &response)
	if err != nil {
		return ServiceKey{}, response.Warnings, err
	}

	return serviceKey, response.Warnings, nil
}

// DeleteServiceKey deletes the specified Service Key. An updated
// service binding is returned only if acceptsIncomplete is true.
func (client *Client) DeleteServiceKey(serviceKeyGUID string, acceptsIncomplete bool) (ServiceKey, Warnings, error) {
	request, err := client.newHTTPRequest(requestOptions{
		RequestName: internal.DeleteServiceKeyRequest,
		URIParams:   map[string]string{"service_key_guid": serviceKeyGUID},
		Query:       url.Values{"accepts_incomplete": {"false"}},
	})
	if err != nil {
		return ServiceKey{}, nil, err
	}

	var response cloudcontroller.Response
	var serviceKey ServiceKey
	if acceptsIncomplete {
		response = cloudcontroller.Response{
			Result: &serviceKey,
		}
	}

	err = client.connection.Make(request, &response)
	return serviceKey, response.Warnings, err
}

// GetServiceKey returns back a service binding with the provided GUID.
func (client *Client) GetServiceKey(guid string) (ServiceKey, Warnings, error) {
	request, err := client.newHTTPRequest(requestOptions{
		RequestName: internal.GetServiceKeyRequest,
		URIParams:   Params{"service_binding_guid": guid},
	})
	if err != nil {
		return ServiceKey{}, nil, err
	}

	var serviceKey ServiceKey
	response := cloudcontroller.Response{
		Result: &serviceKey,
	}

	err = client.connection.Make(request, &response)
	return serviceKey, response.Warnings, err
}

// GetServiceKeys returns back a list of Service Keys based off of the
// provided filters.
func (client *Client) GetServiceKeys(filters ...Filter) ([]ServiceKey, Warnings, error) {
	request, err := client.newHTTPRequest(requestOptions{
		RequestName: internal.GetServiceKeysRequest,
		Query:       ConvertFilterParameters(filters),
	})
	if err != nil {
		return nil, nil, err
	}

	var fullKeysList []ServiceKey
	warnings, err := client.paginate(request, ServiceKey{}, func(item interface{}) error {
		if binding, ok := item.(ServiceKey); ok {
			fullKeysList = append(fullKeysList, binding)
		} else {
			return ccerror.UnknownObjectInListError{
				Expected:   ServiceKey{},
				Unexpected: item,
			}
		}
		return nil
	})

	return fullKeysList, warnings, err
}

// // GetServiceInstanceServiceKeys returns back a list of Service Keys for the provided service instance GUID.
// func (client *Client) GetServiceInstanceServiceKeys(serviceInstanceGUID string) ([]ServiceKey, Warnings, error) {
// 	request, err := client.newHTTPRequest(requestOptions{
// 		RequestName: internal.GetServiceInstanceServiceKeysRequest,
// 		URIParams:   map[string]string{"service_instance_guid": serviceInstanceGUID},
// 	})
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	var fullKeysList []ServiceKey
// 	warnings, err := client.paginate(request, ServiceKey{}, func(item interface{}) error {
// 		if binding, ok := item.(ServiceKey); ok {
// 			fullKeysList = append(fullKeysList, binding)
// 		} else {
// 			return ccerror.UnknownObjectInListError{
// 				Expected:   ServiceKey{},
// 				Unexpected: item,
// 			}
// 		}
// 		return nil
// 	})
//
// 	return fullKeysList, warnings, err
// }
//
// // GetUserProvidedServiceInstanceServiceKeys returns back a list of Service Keys for the provided user provided service instance GUID.
// func (client *Client) GetUserProvidedServiceInstanceServiceKeys(userProvidedServiceInstanceGUID string) ([]ServiceKey, Warnings, error) {
// 	request, err := client.newHTTPRequest(requestOptions{
// 		RequestName: internal.GetUserProvidedServiceInstanceServiceKeysRequest,
// 		URIParams:   map[string]string{"user_provided_service_instance_guid": userProvidedServiceInstanceGUID},
// 	})
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	var fullKeysList []ServiceKey
// 	warnings, err := client.paginate(request, ServiceKey{}, func(item interface{}) error {
// 		if binding, ok := item.(ServiceKey); ok {
// 			fullKeysList = append(fullKeysList, binding)
// 		} else {
// 			return ccerror.UnknownObjectInListError{
// 				Expected:   ServiceKey{},
// 				Unexpected: item,
// 			}
// 		}
// 		return nil
// 	})
//
// 	return fullKeysList, warnings, err
// }
