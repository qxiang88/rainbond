// Copyright (C) 2014-2018 Goodrain Co., Ltd.
// RAINBOND, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package controller

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	validator "github.com/goodrain/rainbond/util/govalidator"

	"github.com/go-chi/chi"
	"github.com/goodrain/rainbond/api/handler"
	"github.com/goodrain/rainbond/api/middleware"
	api_model "github.com/goodrain/rainbond/api/model"
	"github.com/goodrain/rainbond/db"
	dbmodel "github.com/goodrain/rainbond/db/model"
	"github.com/goodrain/rainbond/event"
	httputil "github.com/goodrain/rainbond/util/http"
	"github.com/goodrain/rainbond/worker/discover/model"
	"github.com/jinzhu/gorm"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
)

//StartService StartService
// swagger:operation POST /v2/tenants/{tenant_name}/services/{service_alias}/start  v2 startService
//
// 启动应用
//
// start service
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) StartService(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)

	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	if err := handler.CheckTenantResource(r.Context(), tenant, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}

	sEvent := r.Context().Value(middleware.ContextKey("event")).(*dbmodel.ServiceEvent)
	startStopStruct := &api_model.StartStopStruct{
		TenantID:  tenantID,
		ServiceID: serviceID,
		EventID:   sEvent.EventID,
		TaskType:  "start",
	}
	if err := handler.GetServiceManager().StartStopService(startStopStruct); err != nil {
		httputil.ReturnError(r, w, 500, "get service info error.")
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
	return
}

//StopService StopService
// swagger:operation POST /v2/tenants/{tenant_name}/services/{service_alias}/stop v2 stopService
//
// 关闭应用
//
// stop service
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) StopService(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(middleware.ContextKey("event")).(*dbmodel.ServiceEvent)
	//save event
	defer event.CloseManager()
	startStopStruct := &api_model.StartStopStruct{
		TenantID:  tenantID,
		ServiceID: serviceID,
		EventID:   sEvent.EventID,
		TaskType:  "stop",
	}
	if err := handler.GetServiceManager().StartStopService(startStopStruct); err != nil {
		httputil.ReturnError(r, w, 500, "get service info error.")
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

//RestartService RestartService
// swagger:operation POST /v2/tenants/{tenant_name}/services/{service_alias}/restart v2 restartService
//
// 重启应用
//
// restart service
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) RestartService(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(middleware.ContextKey("event")).(*dbmodel.ServiceEvent)
	defer event.CloseManager()
	startStopStruct := &api_model.StartStopStruct{
		TenantID:  tenantID,
		ServiceID: serviceID,
		EventID:   sEvent.EventID,
		TaskType:  "restart",
	}

	curStatus := t.StatusCli.GetStatus(serviceID)
	if curStatus == "closed" {
		startStopStruct.TaskType = "start"
	}

	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	if err := handler.CheckTenantResource(r.Context(), tenant, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}

	if err := handler.GetServiceManager().StartStopService(startStopStruct); err != nil {
		httputil.ReturnError(r, w, 500, "get service info error.")
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
	return
}

//VerticalService VerticalService
// swagger:operation PUT /v2/tenants/{tenant_name}/services/{service_alias}/vertical v2 verticalService
//
// 应用垂直伸缩
//
// service vertical
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) VerticalService(w http.ResponseWriter, r *http.Request) {
	rules := validator.MapData{
		"container_cpu":    []string{"required"},
		"container_memory": []string{"required"},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(middleware.ContextKey("event")).(*dbmodel.ServiceEvent)
	cpu := int(data["container_cpu"].(float64))
	mem := int(data["container_memory"].(float64))

	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	if err := handler.CheckTenantResource(r.Context(), tenant, service.Replicas*mem); err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}

	verticalTask := &model.VerticalScalingTaskBody{
		TenantID:        tenantID,
		ServiceID:       serviceID,
		EventID:         sEvent.EventID,
		ContainerCPU:    cpu,
		ContainerMemory: mem,
	}
	if err := handler.GetServiceManager().ServiceVertical(verticalTask); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("service vertical error. %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

//HorizontalService HorizontalService
// swagger:operation PUT /v2/tenants/{tenant_name}/services/{service_alias}/horizontal v2 horizontalService
//
// 应用水平伸缩
//
// service horizontal
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) HorizontalService(w http.ResponseWriter, r *http.Request) {
	rules := validator.MapData{
		"node_num": []string{"required"},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(middleware.ContextKey("event")).(*dbmodel.ServiceEvent)
	replicas := int32(data["node_num"].(float64))

	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	if err := handler.CheckTenantResource(r.Context(), tenant, service.ContainerMemory*int(replicas)); err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}

	horizontalTask := &model.HorizontalScalingTaskBody{
		TenantID:  tenantID,
		ServiceID: serviceID,
		EventID:   sEvent.EventID,
		Username:  sEvent.UserName,
		Replicas:  replicas,
	}

	if err := handler.GetServiceManager().ServiceHorizontal(horizontalTask); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

//BuildService BuildService
// swagger:operation POST /v2/tenants/{tenant_name}/services/{service_alias}/build v2 serviceBuild
//
// 应用构建
//
// service build
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) BuildService(w http.ResponseWriter, r *http.Request) {
	var build api_model.ComponentBuildReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &build, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	tenantName := r.Context().Value(middleware.ContextKey("tenant_name")).(string)
	build.TenantName = tenantName
	build.EventID = r.Context().Value(middleware.ContextKey("event_id")).(string)
	if build.ServiceID != serviceID {
		httputil.ReturnError(r, w, 400, "build service id is failure")
		return
	}

	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	if err := handler.CheckTenantResource(r.Context(), tenant, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}

	res, err := handler.GetOperationHandler().Build(&build)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

//BuildList BuildList
func (t *TenantStruct) BuildList(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)

	resp, err := handler.GetServiceManager().ListVersionInfo(serviceID)

	if err != nil {
		logrus.Error("get version info error", err.Error())
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get version info erro, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

//BuildVersionIsExist -
func (t *TenantStruct) BuildVersionIsExist(w http.ResponseWriter, r *http.Request) {
	statusMap := make(map[string]bool)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	buildVersion := chi.URLParam(r, "build_version")
	_, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(buildVersion, serviceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get build version status erro, %v", err))
		return
	}
	if err == gorm.ErrRecordNotFound {
		statusMap["status"] = false
	} else {
		statusMap["status"] = true
	}
	httputil.ReturnSuccess(r, w, statusMap)

}

//DeleteBuildVersion -
func (t *TenantStruct) DeleteBuildVersion(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	buildVersion := chi.URLParam(r, "build_version")
	val, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(buildVersion, serviceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
		return
	}
	if err == gorm.ErrRecordNotFound {

	} else {
		if val.DeliveredType == "slug" && val.FinalStatus == "success" {
			if err := os.Remove(val.DeliveredPath); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return

			}
			if err := db.GetManager().VersionInfoDao().DeleteVersionInfo(val); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return

			}
		}
		if val.FinalStatus == "failure" {
			if err := db.GetManager().VersionInfoDao().DeleteVersionInfo(val); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return
			}
		}
		if val.DeliveredType == "image" {
			if err := db.GetManager().VersionInfoDao().DeleteVersionInfo(val); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return
			}
		}
	}
	httputil.ReturnSuccess(r, w, nil)

}

//UpdateBuildVersion -
func (t *TenantStruct) UpdateBuildVersion(w http.ResponseWriter, r *http.Request) {
	var build api_model.UpdateBuildVersionReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &build, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	buildVersion := chi.URLParam(r, "build_version")
	versionInfo, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(buildVersion, serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("update build version info error, %v", err))
		return
	}
	versionInfo.PlanVersion = build.PlanVersion
	err = db.GetManager().VersionInfoDao().UpdateModel(versionInfo)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("update build version info error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

//BuildVersionInfo -
func (t *TenantStruct) BuildVersionInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		t.DeleteBuildVersion(w, r)
	case "GET":
		t.BuildVersionIsExist(w, r)
	case "PUT":
		t.UpdateBuildVersion(w, r)
	}

}

//GetDeployVersion GetDeployVersion by service
func (t *TenantStruct) GetDeployVersion(w http.ResponseWriter, r *http.Request) {
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(service.DeployVersion, service.ServiceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get build version status erro, %v", err))
		return
	}
	if err == gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 404, fmt.Sprintf("build version do not exist"))
		return
	}
	httputil.ReturnSuccess(r, w, version)
}

//GetManyDeployVersion GetDeployVersion by some service id
func (t *TenantStruct) GetManyDeployVersion(w http.ResponseWriter, r *http.Request) {
	rules := validator.MapData{
		"service_ids": []string{"required"},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	serviceIDs, ok := data["service_ids"].([]interface{})
	if !ok {
		httputil.ReturnError(r, w, 400, fmt.Sprintf("service ids must be a array"))
		return
	}
	var list []string
	for _, s := range serviceIDs {
		list = append(list, s.(string))
	}
	services, err := db.GetManager().TenantServiceDao().GetServiceByIDs(list)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf(err.Error()))
		return
	}
	var versionList []*dbmodel.VersionInfo
	for _, service := range services {
		version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(service.DeployVersion, service.ServiceID)
		if err != nil && err != gorm.ErrRecordNotFound {
			httputil.ReturnError(r, w, 500, fmt.Sprintf("get build version status erro, %v", err))
			return
		}
		versionList = append(versionList, version)
	}
	httputil.ReturnSuccess(r, w, versionList)
}

//DeployService DeployService
func (t *TenantStruct) DeployService(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("trans deploy service")
	w.Write([]byte("deploy service"))
}

//UpgradeService UpgradeService
// swagger:operation POST /v2/tenants/{tenant_name}/services/{service_alias}/upgrade v2 upgradeService
//
// 升级应用
//
// upgrade service
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) UpgradeService(w http.ResponseWriter, r *http.Request) {
	var upgradeRequest api_model.ComponentUpgradeReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &upgradeRequest, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	upgradeRequest.EventID = r.Context().Value(middleware.ContextKey("event_id")).(string)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	if upgradeRequest.ServiceID != serviceID {
		httputil.ReturnError(r, w, 400, "upgrade service id failure")
		return
	}

	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	if err := handler.CheckTenantResource(r.Context(), tenant, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}

	res, err := handler.GetOperationHandler().Upgrade(&upgradeRequest)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

//CheckCode CheckCode
// swagger:operation POST /v2/tenants/{tenant_name}/code-check v2 checkCode
//
// 应用代码检测
//
// check  code
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) CheckCode(w http.ResponseWriter, r *http.Request) {

	var ccs api_model.CheckCodeStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ccs.Body, nil)
	if !ok {
		return
	}
	if ccs.Body.TenantID == "" {
		tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
		ccs.Body.TenantID = tenantID
	}
	ccs.Body.Action = "code_check"
	if err := handler.GetServiceManager().CodeCheck(&ccs); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("task code check error,%v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

//RollBack RollBack
// swagger:operation Post /v2/tenants/{tenant_name}/services/{service_alias}/rollback v2 rollback
//
// 应用版本回滚
//
// service rollback
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//   default:
//     schema:
//       "$ref": "#/responses/commandResponse"
//     description: 统一返回格式
func (t *TenantStruct) RollBack(w http.ResponseWriter, r *http.Request) {
	var rollbackRequest api_model.RollbackInfoRequestStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &rollbackRequest, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	if rollbackRequest.ServiceID != serviceID {
		httputil.ReturnError(r, w, 400, "rollback service id failure")
		return
	}
	rollbackRequest.EventID = r.Context().Value(middleware.ContextKey("event_id")).(string)

	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	service := r.Context().Value(middleware.ContextKey("service")).(*dbmodel.TenantServices)
	if err := handler.CheckTenantResource(r.Context(), tenant, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}

	re := handler.GetOperationHandler().RollBack(rollbackRequest)
	httputil.ReturnSuccess(r, w, re)
	return
}

type limitMemory struct {
	LimitMemory int `json:"limit_memory"`
}

//LimitTenantMemory -
func (t *TenantStruct) LimitTenantMemory(w http.ResponseWriter, r *http.Request) {
	var lm limitMemory
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	err = ffjson.Unmarshal(body, &lm)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	tenant, err := db.GetManager().TenantDao().GetTenantByUUID(tenantID)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	tenant.LimitMemory = lm.LimitMemory
	if err := db.GetManager().TenantDao().UpdateModel(tenant); err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
	}
	httputil.ReturnSuccess(r, w, "success!")

}

//SourcesInfo -
type SourcesInfo struct {
	TenantID        string `json:"tenant_id"`
	AvailableMemory int    `json:"available_memory"`
	Status          bool   `json:"status"`
	MemTotal        int    `json:"mem_total"`
	MemUsed         int    `json:"mem_used"`
	CPUTotal        int    `json:"cpu_total"`
	CPUUsed         int    `json:"cpu_used"`
}

//TenantResourcesStatus tenant resources status
func (t *TenantStruct) TenantResourcesStatus(w http.ResponseWriter, r *http.Request) {

	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	tenant, err := db.GetManager().TenantDao().GetTenantByUUID(tenantID)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	//11ms
	services, err := handler.GetServiceManager().GetService(tenant.UUID)
	if err != nil {
		msg := httputil.ResponseBody{
			Msg: fmt.Sprintf("get service error, %v", err),
		}
		httputil.Return(r, w, 500, msg)
		return
	}

	statsInfo, _ := handler.GetTenantManager().StatsMemCPU(services)

	if tenant.LimitMemory == 0 {
		sourcesInfo := SourcesInfo{
			TenantID:        tenantID,
			AvailableMemory: 0,
			Status:          true,
			MemTotal:        tenant.LimitMemory,
			MemUsed:         statsInfo.MEM,
			CPUTotal:        0,
			CPUUsed:         statsInfo.CPU,
		}
		httputil.ReturnSuccess(r, w, sourcesInfo)
		return
	}
	if statsInfo.MEM >= tenant.LimitMemory {
		sourcesInfo := SourcesInfo{
			TenantID:        tenantID,
			AvailableMemory: tenant.LimitMemory - statsInfo.MEM,
			Status:          false,
			MemTotal:        tenant.LimitMemory,
			MemUsed:         statsInfo.MEM,
			CPUTotal:        tenant.LimitMemory / 4,
			CPUUsed:         statsInfo.CPU,
		}
		httputil.ReturnSuccess(r, w, sourcesInfo)
	} else {
		sourcesInfo := SourcesInfo{
			TenantID:        tenantID,
			AvailableMemory: tenant.LimitMemory - statsInfo.MEM,
			Status:          true,
			MemTotal:        tenant.LimitMemory,
			MemUsed:         statsInfo.MEM,
			CPUTotal:        tenant.LimitMemory / 4,
			CPUUsed:         statsInfo.CPU,
		}
		httputil.ReturnSuccess(r, w, sourcesInfo)
	}
}

//GetServiceDeployInfo get service deploy info
func GetServiceDeployInfo(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(middleware.ContextKey("tenant_id")).(string)
	serviceID := r.Context().Value(middleware.ContextKey("service_id")).(string)
	info, err := handler.GetServiceManager().GetServiceDeployInfo(tenantID, serviceID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, info)
}
