package broker

import (
	"context"
	"errors"
	"fmt"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/constant"
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
)

type InstanceCredentials struct {
	Host     string
	Port     int
	Password string
}

type InstanceCreator interface {
	Create() error
	Destroy() error
	InstanceExists() (bool, error)
}

type InstanceBinder interface {
	Bind() (InstanceCredentials, error)
	Unbind() error
	InstanceExists() (bool, error)
}

type CFBrokerProxy struct {
	InstanceCreators map[string]InstanceCreator
	InstanceBinders  map[string]InstanceBinder
	APIClient        *ccv2.Client
	Logger           lager.Logger
	Catalog          []brokerapi.Service
	SpaceGUID        string
}

func (broker *CFBrokerProxy) Services(context context.Context) ([]brokerapi.Service, error) {
	if broker.Catalog != nil {
		return broker.Catalog, nil
	}

	services, _, err := broker.APIClient.GetServices()

	if err != nil {
		return []brokerapi.Service{}, err
	}

	serviceList := []brokerapi.Service{}

	for _, s := range services {
		planList := broker.getServicePlans(s.GUID)

		service := brokerapi.Service{
			ID:            s.GUID,
			Name:          s.Label,
			Description:   sliceDescription(s.Description),
			Bindable:      true,
			PlanUpdatable: false,
			Plans:         planList,
		}
		serviceList = append(serviceList, service)
	}
	broker.Catalog = serviceList
	return serviceList, nil
}

func (broker *CFBrokerProxy) Provision(context context.Context, instanceID string, serviceDetails brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	if broker.SpaceGUID == "" {
		availableSpace, _, err := broker.APIClient.GetSpaces()
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
		if len(availableSpace) < 1 {
			return brokerapi.ProvisionedServiceSpec{}, errors.New("Available spaces not found")
		}
		broker.SpaceGUID = availableSpace[0].GUID
	}

	serviceInstance, _, err := broker.APIClient.CreateServiceInstance(instanceID,
		serviceDetails.PlanID, broker.SpaceGUID, asyncAllowed, make(map[string]interface{}))

	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	return brokerapi.ProvisionedServiceSpec{
		DashboardURL:  serviceInstance.DashboardURL,
		IsAsync:       serviceInstance.LastOperation.State == constant.LastOperationInProgress,
		OperationData: serviceInstance.GUID,
	}, nil
}

func (broker *CFBrokerProxy) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	firstInstance, err := broker.findFirstServiceInstanceWithName(instanceID)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	instanceToDelete, _, err := broker.APIClient.DeleteServiceInstance(firstInstance.GUID, true)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	return brokerapi.DeprovisionServiceSpec{
		IsAsync:       instanceToDelete.LastOperation.State == constant.LastOperationInProgress,
		OperationData: instanceToDelete.GUID,
	}, nil
}

func (broker *CFBrokerProxy) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	firstInstance, err := broker.findFirstServiceInstanceWithName(instanceID)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	serviceKey, _, err := broker.APIClient.CreateServiceKey(firstInstance.GUID, bindingID, true, make(map[string]interface{}))
	if err != nil {
		return brokerapi.Binding{}, err
	}

	return brokerapi.Binding{
		Credentials: serviceKey.Credentials,
	}, nil
}

func (broker *CFBrokerProxy) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	serviceKeys, _, err := broker.APIClient.GetServiceKeys(ccv2.Filter{
		Type:     "name",
		Operator: ":",
		Values:   []string{bindingID},
	})

	if err != nil {
		return err
	}
	if len(serviceKeys) < 1 {
		return fmt.Errorf("Service key '%s' not found", bindingID)
	}

	_, _, err = broker.APIClient.DeleteServiceKey(serviceKeys[0].GUID, false)
	if err != nil {
		return err
	}

	return nil
}

func (broker *CFBrokerProxy) LastOperation(context context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	serviceInstance, _, err := broker.APIClient.GetServiceInstance(operationData)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}

	return brokerapi.LastOperation{
		State:       brokerapi.LastOperationState(serviceInstance.LastOperation.State),
		Description: serviceInstance.LastOperation.Description}, nil
}

func (broker *CFBrokerProxy) Update(context context.Context, instanceID string, serviceDetails brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, nil
}

func (broker *CFBrokerProxy) findFirstServiceInstanceWithName(serviceInstanceName string) (ccv2.ServiceInstance, error) {
	serviceInstances, _, err := broker.APIClient.GetServiceInstances(ccv2.Filter{
		Type:     "name",
		Operator: ":",
		Values:   []string{serviceInstanceName},
	})

	if err != nil {
		return ccv2.ServiceInstance{}, err
	}
	if len(serviceInstances) < 1 {
		return ccv2.ServiceInstance{}, fmt.Errorf("Service instance with name %s not found", serviceInstanceName)
	}

	return serviceInstances[0], nil
}

func (broker *CFBrokerProxy) getServicePlans(serviceGUID string) []brokerapi.ServicePlan {
	servicePlans, _, _ := broker.APIClient.GetServicePlans(ccv2.Filter{
		Type:     "service_guid",
		Operator: ":",
		Values:   []string{serviceGUID},
	})
	// TODO if err

	plans := []brokerapi.ServicePlan{}
	for _, p := range servicePlans {

		plan := brokerapi.ServicePlan{
			ID:          p.GUID,
			Name:        p.Name,
			Description: sliceDescription(p.Description),
		}
		plans = append(plans, plan)
	}

	return plans
}

func sliceDescription(original string) string {
	if len(original) > 254 {
		return original[:254]
	}

	return original
}
