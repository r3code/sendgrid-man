package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

func (tf templateFileStore) Store(mt mailTemplate, includePlain, overwriteExisting bool, allVersions bool) error {
	if len(mt.Versions) == 0 {
		return fmt.Errorf("No versions for TemplateID=%s", mt.ID)
	}
	for _, tplVer := range mt.Versions {
		if !allVersions && tplVer.Active == 0 {
			fmt.Printf("SKIP: inactive version %s\n", tplVer.ID)
			continue
		}
		fmt.Printf("  Version:\n    ID:    %s\n    Name: '%s'\n    Active: %d\n", tplVer.ID, tplVer.Name, tplVer.Active)

		htmlFileName := filepath.Join(tf.baseDir, fmt.Sprintf("%s.html", mt.Name))
		plainFileName := filepath.Join(tf.baseDir, fmt.Sprintf("%s.txt", mt.Name))
		if allVersions {
			htmlFileName = filepath.Join(tf.baseDir, fmt.Sprintf("%s__%s.html", mt.Name, tplVer.ID))
			plainFileName = filepath.Join(tf.baseDir, fmt.Sprintf("%s__%s.txt", mt.Name, tplVer.ID))
		}

		if _, err := os.Stat(htmlFileName); err == nil {
			fmt.Printf("*** WARN: file exists '%s'\n", htmlFileName)
			if !overwriteExisting {
				fmt.Println("Skip save. Overwrite not allowed")
				continue
			}
			fmt.Println("Overwrite")
		}
		err := ioutil.WriteFile(htmlFileName, []byte(tplVer.HTMLContent), 0644)
		if err != nil {
			return fmt.Errorf("store TemplateID='%s'/VersionID=%s HTML content to file '%s' fail: %w", tplVer.TemplateID, tplVer.ID, htmlFileName, err)
		}
		fmt.Printf("File created '%s'\n", htmlFileName)
		if includePlain {
			err = ioutil.WriteFile(plainFileName, []byte(tplVer.PlainContent), 0644)
			if err != nil {
				return fmt.Errorf("store TemplateID='%s'/VersionID=%s PLAIN content to file '%s' fail: %w", tplVer.TemplateID, tplVer.ID, plainFileName, err)
			}
		}
	}
	return nil
}

func main() {
	var apiKey string
	var baseDir string
	var includePlain bool
	var overwriteExisting bool
	var allVersions bool
	flag.StringVar(&apiKey, "apikey", "", "[required] sendgrid APIKey (not API Key ID!) to access service")
	flag.StringVar(&baseDir, "basedir", "", "base dir where templates are stored")
	flag.BoolVar(&includePlain, "include_plain", false, "export also plain templates (by defult html only)")
	flag.BoolVar(&overwriteExisting, "overwrite", false, "allow overwrite existring files")
	flag.BoolVar(&allVersions, "all", false, "save all versions (by default only active)")
	flag.Parse()
	if strings.TrimSpace(apiKey) == "" {
		fmt.Println("Error: Invalid --apikey value, must be not empty and starts with 'SG.'")
		fmt.Println("apiKey: " + apiKey)
		flag.Usage()
		os.Exit(2)
	}

	if strings.TrimSpace(baseDir) == "" {
		wdPath, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		baseDir = wdPath
	}
	baseDir = filepath.Clean(baseDir)
	fmt.Printf("Export to DIR: %s\n", baseDir)
	fmt.Printf("Export plain: %v\n", includePlain)
	fmt.Printf("Overwrite existing: %v\n", overwriteExisting)
	fmt.Printf("Save all versions: %v\n", allVersions)
	fmt.Println()

	sm := &sendgridman{apiKey: apiKey, host: sendgridHost}
	tl, err := sm.getTemplateList()
	if err != nil {
		fmt.Printf("Error: getTemplateList %s", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Found %d dynamic templates\n", len(tl))

	ts := &templateFileStore{
		baseDir: baseDir,
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
		fmt.Printf("\n%d. TemplateID: %s\n  Name: '%s'\n  Versions: %d\n", i+1, mt.ID, mt.Name, len(mt.Versions))
		if err := ts.Store(mt, includePlain, overwriteExisting, allVersions); err != nil {
			fmt.Printf("ERROR: failed to store template ID=%s to file, %s\n\n", mt.ID, err.Error())
		}
	}
	fmt.Println("Retreive templates data: OK")
	// fmt.Printf("tpl list %+v\n", tl)
}
