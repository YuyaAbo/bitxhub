package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

const (
	ServicePreKey = "service-"

	REGISTERED = 0
)

type ServiceManager struct {
	boltvm.Stub
}

type ServiceInfo struct {
	Id      string          `json:"id"`      // service id
	Name    string          `json:"name"`    // service name
	Index   int64           `json:"index"`   // service index
	Desc    string          `json:"desc"`    // service description
	Address string          `json:"address"` // service
	Items   map[string]Item `json:"items"`   // service entities
	Status  int32           `json:"status"`  // 0 => registered, 1 => approved, -1 => rejected
}

type Item struct {
	Method  string   `json:"method"`   // method desc
	ArgType []string `json:"arg_type"` // method arg type
	ArgName []string `json:"arg_name"` // method arg name
	Example []string `json:"example"`  // method arg example
	Status  int32    `json:"status"`   // -1 => rejected, 1 => approved
}

type auditRecord struct {
	Service    *ServiceInfo `json:"service"`
	IsApproved bool         `json:"is_approved"`
	Desc       string       `json:"desc"`
}

type auditItemRecord struct {
	ServiceId  string `json:"service_id"`
	Item       *Item  `json:"item"`
	IsApproved bool   `json:"is_approved"`
	Desc       string `json:"desc"`
}

func NewServiceMng() agency.Contract {
	return &ServiceManager{}
}

func (sm *ServiceManager) Register(name string, desc string, address string, itemData []byte) *boltvm.Response {
	id := sm.Caller()
	// res := sm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(id))
	// if !res.Ok {
	// 	return res
	// }

	sm.Logger().Info("get appchain success")
	var items map[string]Item
	if err := json.Unmarshal(itemData, &items); err != nil {
		return boltvm.Error(err.Error())
	}

	service := &ServiceInfo{
		Id:      id,
		Name:    name,
		Desc:    desc,
		Address: address,
		Items:   items,
		Status:  REGISTERED,
	}
	ok, value := sm.Query(sm.servicePreKey(id))
	if !ok {
		service.Index = 0
		sm.SetObject(sm.serviceKey(id, 0), service)
		body, err := json.Marshal(service)
		if err != nil {
			return boltvm.Error(err.Error())
		}
		sm.Logger().WithFields(logrus.Fields{
			"id":    id,
			"index": 0,
		}).Info("Service register successfully")
		return boltvm.Success(body)
	}
	index := len(value)
	service.Index = int64(index)
	sm.SetObject(sm.serviceKey(id, int64(index)), service)
	body, err := json.Marshal(service)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	sm.Logger().WithFields(logrus.Fields{
		"id":    id,
		"index": index,
	}).Info("Service register successfully")
	return boltvm.Success(body)
}

func (sm *ServiceManager) Call(data []byte) *boltvm.Response {
	var ibtp pb.IBTP
	err := ibtp.Unmarshal(data)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	res := sm.CrossInvoke(constant.InterchainContractAddr.Address().String(), "HandleIBTP", pb.Bytes(data))
	return res
}

// func (sm *ServiceManager) Update(name string, desc string, itemData []byte) *boltvm.Response {
// 	id := sm.Caller()
// 	ok := sm.Has(sm.serviceKey(id))
// 	if !ok {
// 		return boltvm.Error("register service firstly")
// 	}

// 	service := sm.getServiceInfo(id)

// 	if service.Status == REGISTERED {
// 		return boltvm.Error("this service is being audited")
// 	}

// 	var items map[string]Item
// 	if err := json.Unmarshal(itemData, &items); err != nil {
// 		return boltvm.Error(err.Error())
// 	}
// 	service = &ServiceInfo{
// 		Name:   name,
// 		Desc:   desc,
// 		Items:  items,
// 		Status: service.Status,
// 	}

// 	sm.SetObject(sm.serviceKey(id), service)
// 	return boltvm.Success(nil)
// }

func (sm *ServiceManager) GetServiceInfo(id string, index int64) *boltvm.Response {
	service, err := json.Marshal(sm.getServiceInfo(id, index))
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(service)
}

func (sm *ServiceManager) getServiceInfo(id string, index int64) *ServiceInfo {
	service := &ServiceInfo{}
	sm.GetObject(sm.serviceKey(id, index), service)
	return service
}

func (sm *ServiceManager) ListService() *boltvm.Response {
	ok, value := sm.Query(ServicePreKey)
	if !ok {
		return boltvm.Success(nil)
	}

	ret := make([]*ServiceInfo, 0)
	for _, data := range value {
		service := &ServiceInfo{}
		if err := json.Unmarshal(data, service); err != nil {
			return boltvm.Error(err.Error())
		}
		ret = append(ret, service)
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

func (sm *ServiceManager) DeleteService(id string, index int64) *boltvm.Response {
	if res := sm.IsAdmin(); !res.Ok {
		return res
	}
	sm.Delete(sm.serviceKey(id, index))
	return boltvm.Success([]byte(fmt.Sprintf("delete service:%s", id)))
}

func (sm *ServiceManager) IsAdmin() *boltvm.Response {
	ret := sm.CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", pb.String(sm.Caller()))
	is, err := strconv.ParseBool(string(ret.Result))
	if err != nil {
		return boltvm.Error(fmt.Errorf("judge caller type: %w", err).Error())
	}

	if !is {
		return boltvm.Error("caller is not an admin account")
	}
	return boltvm.Success([]byte("1"))
}

func (sm *ServiceManager) servicePreKey(id string) string {
	return ServicePreKey + id
}

func (sm *ServiceManager) serviceKey(id string, index int64) string {
	return ServicePreKey + id + "-" + strconv.FormatInt(index, 10)
}

func (sm *ServiceManager) auditRecordKey(id string) string {
	return "audit-record-" + id
}

func (sm *ServiceManager) auditItemRecordKey(id string) string {
	return "audit-item-record-" + id
}
