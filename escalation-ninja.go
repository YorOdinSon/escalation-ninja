package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
)

// :x: Failed to create channel: Slack API error: invalid_name_specials - there might be the chance it's importing colons or it might be breaking because the jira is in upper case
func fixChannelName(name string) string {
	name = strings.ToLower(name)              // Convert to lowercase
	name = strings.ReplaceAll(name, "_", "-") // Replace underscores with dashes
	name = strings.ReplaceAll(name, " ", "-") // Replace spaces with dashes
	name = strings.ReplaceAll(name, ".", "-") // Replace dots with dashes
	name = strings.ReplaceAll(name, "@", "-") // Replace @ with dashes
	name = strings.ReplaceAll(name, "#", "-") // Replace # with dashes
	name = strings.ReplaceAll(name, ":", "-") // Replace colons with dashes
	return name
}

// will need the SlackRequest to be handled as an object - I think it'll be easier
type SlackRequest struct {
	Token       string `json:"token"`
	TeamID      string `json:"team_id"`
	TeamDomain  string `json:"team_domain"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	Command     string `json:"command"`
	Text        string `json:"text"`
	ResponseURL string `json:"response_url"`
	TriggerID   string `json:"trigger_id"`
}

// TEMP: Struct for Slack API response
type SlackChannelResponse struct {
	OK      bool `json:"ok"`
	Channel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channel"`
	Error string `json:"error,omitempty"`
}

// creating a function to handle the Slack Channel creation
func createSlackChannel(channelName string) (string, error) {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	fmt.Println("DEBUG - Slack Token:", slackToken) //checking the Token, as I got a "Token not set" error
	if slackToken == "" {
		return "", fmt.Errorf("Slack token not set")
	}

	slackApiUrl := "https://slack.com/api/conversations.create"

	//Requesting payload
	payload := map[string]string{
		"name": channelName,
	}

	//payload to JSON
	payloadBytes, _ := json.Marshal(payload)

	//HTTP request
	req, err := http.NewRequest("POST", slackApiUrl, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}

	//setting headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	//executing the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	//Decode response
	var slackResp SlackChannelResponse
	json.NewDecoder(resp.Body).Decode(&slackResp)

	//printing the response for debugging since I got a "unknown_method" error
	fmt.Println("DEBUG - Slack API Response:", slackResp)

	//Handling the API response
	if !slackResp.OK {
		return "", fmt.Errorf("Slack API error: %s", slackResp.Error)
	}

	log.Printf("✅ Channel Create: #%s (ID: %s)", slackResp.Channel.Name, slackResp.Channel.ID)
	return slackResp.Channel.ID, nil
}

// creating a function to invite the users to the channel - it will require a reintepretation of the userID to be transformed in a tag
func inviteUsersToChannel(channelID string, users []string, commandUserID string) error {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		return fmt.Errorf("Slack token not set")
	}
	inviteApiUrl := "https://slack.com/api/conversations.invite"

	// Ensure the command user is always invited
	users = append(users, commandUserID)

	// ✅ Convert ONLY usernames to Slack user IDs
	userIDs := convertUserTagsToIDs(users)

	// ✅ Add the command user ID directly (since we already have it)
	userIDs = append(userIDs, commandUserID)

	// Remove duplicate user IDs
	userIDSet := make(map[string]bool)
	var uniqueUserIDs []string
	for _, userID := range userIDs {
		if _, exists := userIDSet[userID]; !exists {
			userIDSet[userID] = true
			uniqueUserIDs = append(uniqueUserIDs, userID)
		}
	}
	if len(uniqueUserIDs) == 0 {
		return fmt.Errorf("No valid users to invite")
	}

	//creating comma-separated users IDs for the payload
	payload := map[string]interface{}{
		"channel": channelID,
		"users":   strings.Join(userIDs, ","),
	}

	payloadBytes, _ := json.Marshal(payload)

	//Creating the HTTP request
	req, err := http.NewRequest("POST", inviteApiUrl, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}

	//headers setting
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	//Executing the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//decoding the response
	var slackResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&slackResp)

	//Printing Slack Response - just for Debugging
	fmt.Println("DEBUG - Invite Users Response:", slackResp)

	if ok, exists := slackResp["ok"].(bool); !exists || !ok {
		return fmt.Errorf("Slack API error: %v", slackResp["error"])
	}

	log.Printf("✅ Invited users %v to channel ID: %s", userIDs, channelID)
	return nil

}

// creating the function to convert the @usernames to user IDs
func convertUserTagsToIDs(userTags []string) []string {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		log.Println("Slack token not set")
		return nil
	}

	url := "https://slack.com/api/users.list"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+slackToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("❌ Error fetching user list:", err)
		return nil
	}
	defer resp.Body.Close()

	var slackResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&slackResp)

	if ok, exists := slackResp["ok"].(bool); !exists || !ok {
		log.Println("⚠️ Slack API error:", slackResp["error"])
		return nil
	}

	// Extract users from response
	members, exists := slackResp["members"].([]interface{})
	if !exists {
		log.Println("❌ Error: No members found in workspace.")
		return nil
	}

	// Build a map of username -> user ID
	userMap := make(map[string]string)
	for _, member := range members {
		userData := member.(map[string]interface{})
		username := userData["name"].(string)
		userID := userData["id"].(string)

		userMap[username] = userID
	}

	// Convert `@username` tags to Slack user IDs
	var userIDs []string
	for _, tag := range userTags {
		username := strings.TrimPrefix(tag, "@")
		if userID, exists := userMap[username]; exists {
			userIDs = append(userIDs, userID)
		} else {
			log.Printf("⚠️ Warning: No Slack user found for @%s", username)
		}
	}

	log.Println("DEBUG - Converted User IDs:", userIDs)
	return userIDs
}

/*
After a few tries, the bot still failed - the channel gets created correctly, but the bot fails the invitation

	yorodinson@YorOdinSon ~ % curl -X POST "https://slack.com/api/conversations.invite" \
	     -H "Authorization: Bearer xoxb- \
	     -H "Content-Type: application/json" \
	     -d '{"channel":"C08EE7BU5MH","users":"U12345,U67890"}'

{"ok":false,"error":"not_in_channel","warning":"missing_charset","response_metadata":{"warnings":["missing_charset"]}}%

It turns out, the bot doesn't join the channel so it can't invite people in.
("error":"not_in_channel")

Creating the function to let the Bot join the channel
*/
func joinChannel(channelID string) error {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		return fmt.Errorf("Slack token not set")
	}

	url := "https://slack.com/api/conversations.join"

	payload := map[string]string{
		"channel": channelID,
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var slackResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&slackResp)

	// Debugging: Print response
	fmt.Println("DEBUG - Join Channel Response:", slackResp)

	if ok, exists := slackResp["ok"].(bool); !exists || !ok {
		return fmt.Errorf("Slack API error: %v", slackResp["error"])
	}

	log.Printf("✅ Bot joined channel ID: %s", channelID)
	return nil
}

// creating a function to handle the Slack Request
/*
old handleSlackCommand

func handleSlackCommand(w http.ResponseWriter, r *http.Request) {

	//first: Error handling the httpRequest
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "❌ Failed to parse request!", http.StatusBadRequest)
		fmt.Print("Sorry dude! 😩")
		return
	}

	slackReq := SlackRequest{
		Token:       r.FormValue("token"),
		TeamID:      r.FormValue("team_id"),
		ChannelID:   r.FormValue("channel_id"),
		TeamDomain:  r.FormValue("team_domain"),
		ChannelName: r.FormValue("channel_name"),
		UserID:      r.FormValue("user_id"),
		UserName:    r.FormValue("user_name"),
		Command:     r.FormValue("command"),
		Text:        r.FormValue("text"),
		ResponseURL: r.FormValue("response_url"),
		TriggerID:   r.FormValue("trigger_id"),
	}

	//Check #1 - making sure it's an expected command!
	if !(slackReq.Command == "/ninjaescal" || slackReq.Command == "/ninjaescalate") {
		http.Error(w, "❌ Invalid Command", http.StatusForbidden)
		fmt.Print("❌ Invalid Command - Start your request with '/ninjaescal' or '/ninjaescalate'")
		return
	}

	//Check #2 - Processing the command arguments to see if a case link is provided
	args := strings.TrimSpace(slackReq.Text)
	if args == "" {
		fmt.Fprintln(w, "⚠ Please provide a case link!")
		return
	}

	//Responding to Slack
	response := fmt.Sprintf("Creating escalation channel for: %s", args)
	fmt.Fprintln(w, response)

	log.Printf("Received command from %s: %s", slackReq.UserName, args)

} //function handleSlackCommand end
*/

/*
parsing a little bit - I'll be expecting a string like

/ninjaescal case: https://<link-to-the-case> client: <client-name> invite: @<user.tag1> @<user.tag2>...

So I'll need to parse everything that comes after "case:" for the case URL from which I'll be creating the case number, then after "client: " for a <client-name>, and lastly after "invite:" for the invitee
*/

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

	jiraCase := extractJira(caseURL)
	channelName := fmt.Sprintf("tmp-%s-escalation-%s", clientName, jiraCase)

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
