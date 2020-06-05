package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sendgrid/rest"
	sendgrid "github.com/sendgrid/sendgrid-go"
)

const sendgridHost = "https://api.sendgrid.com"

type sendgridman struct {
	host   string // a sendgrid service hostname
	apiKey string // must be exactly 39 symbols
}

type templateInfo struct {
	ID       string                `json:"id"`
	Name     string                `json:"name"`
	Versions []templateVersionInfo `json:"versions"`
}

type templateVersionInfo struct {
	ID         string `json:"id"`
	TemplateID string `json:"template_id"`
	Active     uint   `json:"active"`
	Name       string `json:"name"`
	UpdatedAt  string `json:"updated_at"`
	Editor     string `json:"editor"`
}

// see https://sendgrid.com/docs/API_Reference/Web_API_v3/Transactional_Templates/templates.html#-GET
type mailTemplate struct {
	ID       string                `json:"id"`
	Name     string                `json:"name"`
	Versions []mailTemplateVersion `json:"versions"`
}

// see https://sendgrid.com/docs/API_Reference/Web_API_v3/Transactional_Templates/templates.html#-GET
type mailTemplateVersion struct {
	templateVersionInfo

	Subject      string `json:"subject"`
	HTMLContent  string `json:"html_content"`
	PlainContent string `json:"plain_content"`
}

// returns a list of dynamic templates descriptions (without a template code and test data)
func (s sendgridman) getTemplateList() (list []templateInfo, err error) {
	request := sendgrid.GetRequest(s.apiKey, "/v3/templates", s.host)
	request.Method = "GET"
	queryParams := make(map[string]string)
	queryParams["generations"] = "dynamic" // only dynamic templates
	request.QueryParams = queryParams
	response, err := sendgrid.API(request)
	if err != nil {
		return list, fmt.Errorf("load templates fail: %w", err)
	}
	// TODO: check status code?
	// fmt.Println(response.StatusCode)
	// fmt.Println(response.Headers)

	type templateList struct {
		Templates []templateInfo `json:"templates"`
	}
	tl := &templateList{}
	err = json.Unmarshal([]byte(response.Body), tl)
	if err != nil {
		return list, fmt.Errorf("parse templates json fail: %w", err)
	}
	return tl.Templates, nil

}

func (s sendgridman) getTemplate(templateID string) (template mailTemplate, err error) {
	request := sendgrid.GetRequest(s.apiKey, fmt.Sprintf("/v3/templates/%s", templateID), s.host)
	request.Method = "GET"
	var response *rest.Response
	response, err = sendgrid.API(request)
	if err != nil {
		return template, fmt.Errorf("load template ID='%s' fail: %w", templateID, err)
	}
	// TODO: check status code?
	// fmt.Println(response.StatusCode)
	// fmt.Println(response.Headers)
	err = json.Unmarshal([]byte(response.Body), &template)
	if err != nil {
		return template, fmt.Errorf("parse template ID='%s' fail: %w", templateID, err)
	}
	return template, nil
}

// func printTemplateList()

func main() {
	var apiKey string
	flag.StringVar(&apiKey, "apikey", "", "a sendgrid APIKey (not API Key ID!) to access service")

	// if len(apiKey) != 39 {
	// 	fmt.Println("Error: Invalid apikey len, must be 39 symbols exactly")
	// 	os.Exit(2)
	// }
	flag.Parse()
	sm := &sendgridman{apiKey: apiKey, host: sendgridHost}
	tl, err := sm.getTemplateList()
	if err != nil {
		fmt.Printf("Error: getTemplateList %s", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Found %d dynamic templates\n", len(tl))
	fmt.Println("Retreive templates data")
	for i, tplInfo := range tl {
		var tpl mailTemplate
		tpl, err = sm.getTemplate(tplInfo.ID)
		if err != nil {
			fmt.Printf("Error: failed to retreive template data ID=%s, %s", tplInfo.ID, err.Error())
			os.Exit(1)
		}
		fmt.Printf("%d. Template ID=%s\n %+v\n\n", i, tplInfo.ID, tpl)
	}
	fmt.Println("Retreive templates data: OK")
	// fmt.Printf("tpl list %+v\n", tl)
}
