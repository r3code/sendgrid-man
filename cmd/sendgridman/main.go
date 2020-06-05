package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

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

type templateFileStore struct {
	baseDir string
}

func (tf templateFileStore) Store(mt mailTemplate) error {
	htmlFileName := path.Join(tf.baseDir, mt.Name+".html")
	plainFileName := path.Join(tf.baseDir, mt.Name+".txt")
	if len(mt.Versions) == 0 {
		fmt.Printf("No versions for TemplateID=%s\n", mt.ID)
	}
	fmt.Printf("Saving versions for TemplateID=%s '%s'\n", mt.ID, mt.Name)
	for _, tplVer := range mt.Versions {
		fmt.Printf("Saving version: %s named '%s'\n", tplVer.ID, tplVer.Name)
		err := ioutil.WriteFile(htmlFileName, []byte(tplVer.HTMLContent), 0644)
		if err != nil {
			return fmt.Errorf("store TemplateID='%s'/VersionID=%s HTML content to file '%s' fail: %w", tplVer.TemplateID, tplVer.ID, htmlFileName, err)
		}
		err = ioutil.WriteFile(plainFileName, []byte(tplVer.PlainContent), 0644)
		if err != nil {
			return fmt.Errorf("store TemplateID='%s'/VersionID=%s PLAIN content to file '%s' fail: %w", tplVer.TemplateID, tplVer.ID, plainFileName, err)
		}
	}
	return nil
}

func main() {
	var apiKey string
	var baseDir string
	flag.StringVar(&apiKey, "apikey", "", "sendgrid APIKey (not API Key ID!) to access service")
	flag.StringVar(&baseDir, "basedir", "", "base dir where templates are stored")
	flag.Parse()
	if strings.TrimSpace(apiKey) == "" {
		fmt.Println("Error: Invalid --apikey value, must be not empty and starts with 'SG.'")
		fmt.Println("apiKey: " + apiKey)
		flag.Usage()
		os.Exit(2)
	}
	if strings.TrimSpace(baseDir) == "" {
		fmt.Println("Error: Invalid path for --basedir")
		fmt.Println("baseDir: " + baseDir)
		flag.Usage()
		os.Exit(2)
	}

	sm := &sendgridman{apiKey: apiKey, host: sendgridHost}
	tl, err := sm.getTemplateList()
	if err != nil {
		fmt.Printf("Error: getTemplateList %s", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Found %d dynamic templates\n", len(tl))

	ts := &templateFileStore{
		baseDir: path.Clean(baseDir),
	}

	fmt.Println("Retreive templates data")
	for i, tplInfo := range tl {
		var mt mailTemplate
		mt, err = sm.getTemplate(tplInfo.ID)
		if err != nil {
			fmt.Printf("Error: failed to retreive template data ID=%s, %s", tplInfo.ID, err.Error())
			os.Exit(1)
		}
		// fmt.Printf("%d. Template ID=%s\n %+v\n\n", i, tplInfo.ID, tpl)
		fmt.Printf("%d. Template ID=%s '%s' \n", i, tplInfo.ID, tplInfo.Name)
		if err := ts.Store(mt); err != nil {
			fmt.Printf("Error: failed to store template ID=%s to file, %s", tplInfo.ID, err.Error())
			os.Exit(1)
		}
	}
	fmt.Println("Retreive templates data: OK")
	// fmt.Printf("tpl list %+v\n", tl)
}
