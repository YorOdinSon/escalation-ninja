package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
)

// :x: Failed to create channel: Slack API error: invalid_name_specials - there might be the chance it's importing colons or it might be breaking because the jira is in upper case

func parseCommand(input string) (string, string, []string) {
	caseRegex := regexp.MustCompile(`case:\s*(https?://[^\s]+)`)
	clientRegex := regexp.MustCompile(`client:\s*(\S+)`)
	inviteRegex := regexp.MustCompile(`invite:\s*(.*)`)

	caseMatch := caseRegex.FindStringSubmatch(input)
	clientMatch := clientRegex.FindStringSubmatch(input)
	inviteMatch := inviteRegex.FindStringSubmatch(input)

	//creating the variables I'm going to return, and that will contain the outcome of the regex formulas!
	caseURL := ""
	clientName := ""
	inviteUsers := []string{}

	if len(caseMatch) > 1 {
		caseURL = caseMatch[1]
	}
	if len(clientMatch) > 1 {
		clientName = clientMatch[1]
	}
	if len(inviteMatch) > 1 {
		inviteUsers = strings.Fields(inviteMatch[1])
	}

	//returning the variables - I'll handle the cases when it's empty

	return caseURL, clientName, inviteUsers
}

// extracting the Jira case number
func extractJira(caseURL string) string {
	jiraRegex := regexp.MustCompile(`browse/(\S+)`)
	match := jiraRegex.FindStringSubmatch(caseURL)

	if len(match) > 1 {
		return match[1] //this should extract the jira case number
	}
	return "no-case"
}

// handling the Slack Command
func handleSlackCommand(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	slackReq := SlackRequest{
		Command: r.FormValue("command"),
		Text:    r.FormValue("text"),
		UserID:  r.FormValue("user_id"), // Capture the user who ran the command
	}

	if !(slackReq.Command == "/ninjaescal" || slackReq.Command == "/ninjaescalate") {
		http.Error(w, "❌ Invalid Command", http.StatusForbidden)
		return
	}

	caseURL, clientName, inviteUsers := parseCommand(slackReq.Text)

	if caseURL == "" {
		fmt.Fprintln(w, "⚠️ Missing required fields: `case:`")
		return
	}
	if clientName == "" {
		fmt.Fprintln(w, "⚠️ Missing required fields: `client:`")
		return
	}
	//working out the case AND the jiraAPIOutcome
	jiraCase := extractJira(caseURL)

	//API outcome ingesting

	jiraIssue := getFromJira(caseURL)

	channelName := fmt.Sprintf("escalation-%s-%s-temp", jiraCase, clientName)

	// Fix invalid characters
	channelName = fixChannelName(channelName)

	// Create Slack channel
	channelID, err := createSlackChannel(channelName)
	if err != nil {
		fmt.Fprintf(w, "❌ Failed to create channel: %s\n", err)
		return
	}

	// ✅ Force the bot to join the channel
	err = joinChannel(channelID)
	if err != nil {
		fmt.Fprintf(w, "⚠️ Channel created, but bot failed to join: %s\n", err)
		return
	}

	//✅ First I'll create the message, then I'll invite Users in
	err = sendSlackMessage(channelID, jiraIssue, caseURL)
	if err != nil {
		fmt.Fprintf(w, "⚠️ Channel created, bot joined, but failed to send message: %s\n", err)
	} else {
		fmt.Fprintf(w, "✅ message sent!")
	}

	// ✅ Always invite the command user along with any specified users
	err = inviteUsersToChannel(channelID, inviteUsers, slackReq.UserID)
	if err != nil {
		fmt.Fprintf(w, "⚠️ Channel created, bot joined, but failed to invite users: %s\n", err)
	} else {
		fmt.Fprintf(w, "✅ Channel Created: #%s (ID: %s), bot joined, and users invited!\n", channelName, channelID)
	}
}

// main contains the webserver answering the command and the Slack Commands sent
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/slack/command", handleSlackCommand).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
