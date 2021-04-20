// Copyright 2013 Matthew Baird
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tableau4go

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const content_type_header = "Content-Type"
const content_length_header = "Content-Length"
const auth_header = "X-Tableau-Auth"
const application_xml_content_type = "application/xml"
const POST = "POST"
const GET = "GET"
const DELETE = "DELETE"

var ErrDoesNotExist = errors.New("Does Not Exist")

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Sign_In%3FTocPath%3DAPI%2520Reference%7C_____51
func (api *API) Signin(username, password string, contentUrl string, userIdToImpersonate string) error {
	url := fmt.Sprintf("%s/api/%s/auth/signin", api.Server, api.Version)
	credentials := Credentials{Name: username, Password: password}
	if len(userIdToImpersonate) > 0 {
		credentials.Impersonate = &User{ID: userIdToImpersonate}
	}
	siteName := contentUrl
	// this seems to have changed. If you are looking for the default site, you must pass
	// blank
	if api.OmitDefaultSiteName {
		if contentUrl == api.DefaultSiteName {
			siteName = ""
		}
	}
	credentials.Site = &Site{ContentUrl: siteName}
	request := SigninRequest{Request: credentials}
	signInXML, err := request.XML()
	if err != nil {
		return err
	}
	payload := string(signInXML)
	headers := make(map[string]string)
	headers[content_type_header] = application_xml_content_type
	retval := AuthResponse{}
	err = api.makeRequest(url, POST, []byte(payload), &retval, headers, connectTimeOut, readWriteTimeout)
	if err == nil {
		api.AuthToken = retval.Credentials.Token
	}
	return err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Sign_Out%3FTocPath%3DAPI%2520Reference%7C_____52
func (api *API) Signout() error {
	url := fmt.Sprintf("%s/api/%s/auth/signout", api.Server, api.Version)
	headers := make(map[string]string)
	headers[content_type_header] = application_xml_content_type
	err := api.makeRequest(url, POST, nil, nil, headers, connectTimeOut, readWriteTimeout)
	return err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Server_Info%3FTocPath%3DAPI%2520Reference%7C__
func (api *API) ServerInfo() (ServerInfo, error) {
	// this call only works on apiVersion 2.4 and up
	url := fmt.Sprintf("%s/api/%s/serverinfo", api.Server, "2.4")
	headers := make(map[string]string)
	retval := ServerInfoResponse{}
	err := api.makeRequest(url, GET, nil, &retval, headers, connectTimeOut, readWriteTimeout)
	return retval.ServerInfo, err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySites() ([]Site, error) {
	url := fmt.Sprintf("%s/api/%s/sites/", api.Server, api.Version)
	headers := make(map[string]string)
	retval := QuerySitesResponse{}
	err := api.makeRequest(url, GET, nil, &retval, headers, connectTimeOut, readWriteTimeout)
	return retval.Sites.Sites, err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySite(siteID string, includeStorage bool) (Site, error) {
	url := fmt.Sprintf("%s/api/%s/sites/%s", api.Server, api.Version, siteID)
	if includeStorage {
		url += fmt.Sprintf("?includeStorage=%v", includeStorage)
	}
	return api.querySite(url)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySiteByName(name string, includeStorage bool) (Site, error) {
	return api.querySiteByKey("name", name, includeStorage)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySiteByContentUrl(contentUrl string, includeStorage bool) (Site, error) {
	return api.querySiteByKey("contentUrl", contentUrl, includeStorage)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) querySiteByKey(key, value string, includeStorage bool) (Site, error) {
	url := fmt.Sprintf("%s/api/%s/sites/%s?key=%s", api.Server, api.Version, value, key)
	if includeStorage {
		url += fmt.Sprintf("&includeStorage=%v", includeStorage)
	}
	return api.querySite(url)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) querySite(url string) (Site, error) {
	headers := make(map[string]string)
	retval := QuerySiteResponse{}
	err := api.makeRequest(url, GET, nil, &retval, headers, connectTimeOut, readWriteTimeout)
	return retval.Site, err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_User_On_Site%3FTocPath%3DAPI%2520Reference%7C_____47
func (api *API) QueryUserOnSite(siteId, userId string) (User, error) {
	url := fmt.Sprintf("%s/api/%s/sites/%s/users/%s", api.Server, api.Version, siteId, userId)
	headers := make(map[string]string)
	retval := QueryUserOnSiteResponse{}
	err := api.makeRequest(url, GET, nil, &retval, headers, connectTimeOut, readWriteTimeout)
	return retval.User, err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Projects%3FTocPath%3DAPI%2520Reference%7C_____38
func (api *API) QueryProjects(siteId string) ([]Project, error) {
	url := fmt.Sprintf("%s/api/%s/sites/%s/projects", api.Server, api.Version, siteId)
	headers := make(map[string]string)
	retval := QueryProjectsResponse{}
	err := api.makeRequest(url, GET, nil, &retval, headers, connectTimeOut, readWriteTimeout)
	return retval.Projects.Projects, err
}

func (api *API) GetProjectByName(siteId, name string) (Project, error) {
	projects, err := api.QueryProjects(siteId)
	if err != nil {
		return Project{}, err
	}
	for _, project := range projects {
		if project.Name == name {
			return project, nil
		}
	}
	return Project{}, fmt.Errorf("Project Named '%s' Not Found", name)
}

func (api *API) GetProjectByID(siteId, ID string) (Project, error) {
	projects, err := api.QueryProjects(siteId)
	if err != nil {
		return Project{}, err
	}
	for _, project := range projects {
		if project.ID == ID {
			return project, nil
		}
	}
	return Project{}, fmt.Errorf("Project with ID '%s' Not Found", ID)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Datasources%3FTocPath%3DAPI%2520Reference%7C_____33
func (api *API) QueryDatasources(siteId string) ([]Datasource, error) {
	url := fmt.Sprintf("%s/api/%s/sites/%s/datasources", api.Server, api.Version, siteId)
	headers := make(map[string]string)
	retval := QueryDatasourcesResponse{}
	err := api.makeRequest(url, GET, nil, &retval, headers, connectTimeOut, readWriteTimeout)
	return retval.Datasources.Datasources, err
}

func (api *API) GetSiteID(siteName string) (string, error) {
	site, err := api.QuerySiteByName(siteName, false)
	if err != nil {
		return "", err
	}
	return site.ID, err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Create_Project%3FTocPath%3DAPI%2520Reference%7C_____14
//POST /api/api-version/sites/site-id/projects
func (api *API) CreateProject(siteId string, project Project) (*Project, error) {
	url := fmt.Sprintf("%s/api/%s/sites/%s/projects", api.Server, api.Version, siteId)
	createProjectRequest := CreateProjectRequest{Request: project}
	xmlRep, err := createProjectRequest.XML()
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	headers[content_type_header] = application_xml_content_type
	createProjectResponse := CreateProjectResponse{}
	err = api.makeRequest(url, POST, xmlRep, &createProjectResponse, headers, connectTimeOut, readWriteTimeout)
	return &createProjectResponse.Project, err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Publish_Datasource%3FTocPath%3DAPI%2520Reference%7C_____31
func (api *API) PublishTDS(siteId string, tdsMetadata Datasource, fullTds string, overwrite bool) (retval *Datasource, err error) {
	return api.publishDatasource(siteId, tdsMetadata, fullTds, "tds", overwrite)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Publish_Datasource%3FTocPath%3DAPI%2520Reference%7C_____31
func (api *API) publishDatasource(siteId string, tdsMetadata Datasource, datasource string, datasourceType string, overwrite bool) (retval *Datasource, err error) {
	url := fmt.Sprintf("%s/api/%s/sites/%s/datasources?datasourceType=%s&overwrite=%v", api.Server, api.Version, siteId, datasourceType, overwrite)
	//payload := fmt.Sprintf("--%s\r\n", api.Boundary)
	payload += "Content-Disposition: name=\"request_payload\"\r\n"
	payload += "Content-Type: text/xml\r\n"
	payload += "\r\n"
	tdsRequest := DatasourceCreateRequest{Request: tdsMetadata}
	xmlRepresentation, err := tdsRequest.XML()
	if err != nil {
		return retval, err
	}
	payload += string(xmlRepresentation)
	//payload += fmt.Sprintf("\r\n--%s\r\n", api.Boundary)
	payload += fmt.Sprintf("Content-Disposition: name=\"tableau_datasource\"; filename=\"%s.tds\"\r\n", tdsMetadata.Name)
	payload += "Content-Type: application/octet-stream\r\n"
	payload += "\r\n"
	payload += datasource
	//payload += fmt.Sprintf("\r\n--%s--\r\n", api.Boundary)
	headers := make(map[string]string)
	//headers[content_type_header] = fmt.Sprintf("multipart/mixed; boundary=%s", api.Boundary)
	err = api.makeRequest(url, POST, []byte(payload), retval, headers, connectTimeOut, readWriteTimeout)
	return retval, err
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Datasource%3FTocPath%3DAPI%2520Reference%7C_____15
func (api *API) DeleteDatasource(siteId string, datasourceId string) error {
	url := fmt.Sprintf("%s/api/%s/sites/%s/datasources/%s", api.Server, api.Version, siteId, datasourceId)
	return api.delete(url)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Project%3FTocPath%3DAPI%2520Reference%7C_____17
func (api *API) DeleteProject(siteId string, projectId string) error {
	url := fmt.Sprintf("%s/api/%s/sites/%s/projects/%s", api.Server, api.Version, siteId, projectId)
	return api.delete(url)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Project%3FTocPath%3DAPI%2520Reference%7C_____17
func (api *API) DeleteSite(siteId string) error {
	url := fmt.Sprintf("%s/api/%s/sites/%s", api.Server, api.Version, siteId)
	return api.delete(url)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Site%3FTocPath%3DAPI%2520Reference%7C_____19
func (api *API) DeleteSiteByName(name string) error {
	return api.deleteSiteByKey("name", name)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Site%3FTocPath%3DAPI%2520Reference%7C_____19
func (api *API) DeleteSiteByContentUrl(contentUrl string) error {
	return api.deleteSiteByKey("contentUrl", contentUrl)
}

//http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Site%3FTocPath%3DAPI%2520Reference%7C_____19
func (api *API) deleteSiteByKey(key string, value string) error {
	url := fmt.Sprintf("%s/api/%s/sites/%s?key=%s", api.Server, api.Version, value, key)
	return api.delete(url)
}

func (api *API) delete(url string) error {
	headers := make(map[string]string)
	return api.makeRequest(url, DELETE, nil, nil, headers, connectTimeOut, readWriteTimeout)
}

func (api *API) makeRequest(requestUrl string, method string, payload []byte, result interface{}, headers map[string]string,
	cTimeout time.Duration, rwTimeout time.Duration) error {
	var debug = true
	if debug {
		fmt.Printf("%s:%v\n", method, requestUrl)
		if payload != nil {
			fmt.Printf("%v\n", string(payload))
		}
	}
	client := DefaultTimeoutClient()
	var req *http.Request
	if len(payload) > 0 {
		var httpErr error
		req, httpErr = http.NewRequest(strings.TrimSpace(method), strings.TrimSpace(requestUrl), bytes.NewBuffer(payload))
		if httpErr != nil {
			return httpErr
		}
		req.Header.Add(content_length_header, strconv.Itoa(len(payload)))
	} else {
		var httpErr error
		req, httpErr = http.NewRequest(strings.TrimSpace(method), strings.TrimSpace(requestUrl), nil)
		if httpErr != nil {
			return httpErr
		}
	}
	if headers != nil {
		for header, headerValue := range headers {
			req.Header.Add(header, headerValue)
		}
	}
	if len(api.AuthToken) > 0 {
		if debug {
			fmt.Printf("%s:%s\n", auth_header, api.AuthToken)
		}
		req.Header.Add(auth_header, api.AuthToken)
	}
	var httpErr error
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		return httpErr
	}
	defer resp.Body.Close()
	body, readBodyError := ioutil.ReadAll(resp.Body)
	if debug {
		fmt.Printf("t4g Response:%v\n", string(body))
	}
	if readBodyError != nil {
		return readBodyError
	}
	if resp.StatusCode == 404 {
		return ErrDoesNotExist
	}
	if resp.StatusCode >= 300 {
		tErrorResponse := ErrorResponse{}
		err := xml.Unmarshal(body, &tErrorResponse)
		if err != nil {
			return err
		}
		return tErrorResponse.Error
	}
	if result != nil {
		// else unmarshall to the result type specified by caller
		err := xml.Unmarshal(body, &result)
		if err != nil {
			return err
		}
	}
	return nil
}
