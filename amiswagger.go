// Package amiswagger
package amiswagger

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/swaggest/openapi-go/openapi3"
	"gitlab.com/ptami_lib/api/v2"
)

var reflector openapi3.Reflector

type Authorization struct {
	Authorization string `header:"Authorization"`
}

type Handler struct {
	api.Handler
	Summary       string
	Request       interface{}
	Response      []OpenApiResponse
	Authorization *string `json:"authorization"`
}

func NewOpenApiHandler(handler api.Handler, summary string, req interface{}, resp []OpenApiResponse) Handler {
	return Handler{
		Handler:  handler,
		Summary:  summary,
		Request:  req,
		Response: resp,
	}
}

type Parameter struct {
	Title       string
	Description string
	Filename    string
	Servers     []openapi3.Server
}

func NewOpenApiParams(title string, description string, filename string, servers []openapi3.Server) Parameter {
	return Parameter{
		Title:       title,
		Description: description,
		Filename:    filename,
		Servers:     servers,
	}
}

type ErrorData struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type ResponseData struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type ResponseDataList struct {
	Error   string  `json:"error"`
	Message string  `json:"message,omitempty"`
	Total   *uint64 `json:"total,omitempty"`
}

type OpenApiResponse struct {
	Code   int
	Output interface{}
}

func selectMethod(req string) (method string, err error) {

	switch req {
	case "GET":
		method = http.MethodGet
	case "PUT":
		method = http.MethodPut
	case "DELETE":
		method = http.MethodDelete
	case "PATCH":
		method = http.MethodPatch
	case "POST":
		method = http.MethodPost
	default:
		err = errors.New("method not found")
	}

	if err != nil {
		return
	}

	return
}

func PrintJson(models interface{}) {
	var jsonData []byte
	var err error
	jsonData, err = json.MarshalIndent(models, "", "\t")
	//jsonData, err = json.Marshal(models)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(string(jsonData))
}

func StructDetail(str interface{}) {
	var metadata = reflect.TypeOf(str)
	for i := 0; i < metadata.NumField(); i++ {
		fmt.Printf("Field %s, Type %s, Tags %s\n", metadata.Field(i).Name, metadata.Field(i).Type, metadata.Field(i).Tag)
	}
}

func GenerateOpenApi(params Parameter, listHandler []Handler) (err error) {
	reflector = openapi3.Reflector{}
	reflector.Spec = &openapi3.Spec{Openapi: "3.0.3"}

	reflector.Spec.Info.
		WithTitle(params.Title).
		WithVersion("1.2.3").
		WithDescription(params.Description)

	reflector.Spec.Servers = params.Servers

	for _, request := range listHandler {
		var method string

		api := openapi3.Operation{
			Summary: aws.String(fmt.Sprintf("[%s][%s] %s", request.Method, request.Resource, request.Summary)),
		}

		method, err = selectMethod(request.Method)
		if nil != err {
			return
		}

		err = reflector.SetRequest(&api, Authorization{}, method)
		if err != nil {
			return
		}

		err = reflector.SetRequest(&api, request.Request, method)
		if err != nil {
			return
		}

		if len(request.Response) < 1 {
			err = errors.New("response cannot be null")
			if err != nil {
				return
			}
		}

		fmt.Println(request.Method, " ", request.Resource)
		for _, item := range request.Response {
			err = reflector.SetJSONResponse(&api, item.Output, item.Code)
			if err != nil {
				return
			}
		}

		err = reflector.Spec.AddOperation(method, request.Resource, api)

		if err != nil {
			return
		}
	}
	//
	var schema []byte
	schema, err = reflector.Spec.MarshalYAML()
	if err != nil {
		return
	}

	err = os.WriteFile(params.Filename+".yml", schema, 0644)
	if err != nil {
		err = fmt.Errorf("unable to write data into the file: %s", err.Error())
	}

	log.Println(string(schema))

	log.Println("Success exported !")
	return
}
