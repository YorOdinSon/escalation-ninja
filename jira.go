package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
)

//based on the jsonoutcome.json, I'll try to create the struct for the fields I'm gonna need.

type JiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary  string `json:"summary"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
		Description struct {
			Content []interface{} `json:"content"` // (optional, if you parse descriptions later)
		} `json:"description"`
	} `json:"fields"`
}

// will get the domain and the case number and make it an API call to Jira
func getAPIJira(jiraURL string) (jiraAPIURL string, err error) {
	re := regexp.MustCompile(`(https?://[^/]+)/browse/([^/]+)`)
	matches := re.FindStringSubmatch(jiraURL)

	if len(matches) != 3 {
		return "", fmt.Errorf("invalid Jira URL format: %s", jiraURL)
	}

	jiraAPIURL = matches[1] + "/rest/api/3/issue/" + matches[2]
	return jiraAPIURL, nil
}

func getFromJira(jiraURL string) JiraIssue {
	fmt.Printf("😎 The jiraURL is %s\n", jiraURL)
	jiraAPI, _ := getAPIJira(jiraURL)
	fmt.Printf("😌 the API for your Jira is:\n %s ✅", jiraAPI)

	jiraUser := os.Getenv("JIRA_EMAIL")
	if jiraUser == "" {
		fmt.Printf("No email Found, for use")
	}
	jiraToken := os.Getenv("JIRA_API_TOKEN")

	if jiraToken == "" {
		fmt.Printf("No Token Found, for user: %s", jiraUser)
	}

	//now that everything is ok, I'll make the HTTP Request - it's a get and the header will contain the jiraUser and the jiraToken

	req, err := http.NewRequest("GET", jiraAPI, nil)
	if err != nil {
		log.Fatal(err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(jiraUser + ":" + jiraToken))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var jiraIssue JiraIssue
	if err := json.Unmarshal(body, &jiraIssue); err != nil {
		log.Fatal(err)
	}

	return jiraIssue
}
