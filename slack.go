package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

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

// looking at the documentation, the JSON shold be this easy
type SlackMessage struct {
	Channel string        `json:"channel"`
	Text    string        `json:"text"`
	Blocks  []interface{} `json:"blocks,omitempty"` // Optional for rich formatting
}

// to pin, I'll need this struct as it capturest the timestamp of the sent message to then use that as a key to pin it.
type SlackMessageResponse struct {
	Ok      bool   `json:"ok"`
	Channel string `json:"channel"`
	Ts      string `json:"ts"` // ✅ Capture the timestamp of the sent message
}

func pinSlackMessage(channelID, timestamp string) error {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		return fmt.Errorf("missing SLACK_BOT_TOKEN")
	}

	slackURL := "https://slack.com/api/pins.add"

	payload := map[string]string{
		"channel":   channelID,
		"timestamp": timestamp,
	}

	jsonBody, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", slackURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("DEBUG - Pin API Response:", string(body)) // 🔍 Log full response

	var response map[string]interface{}
	json.Unmarshal(body, &response)

	if ok, exists := response["ok"].(bool); !exists || !ok {
		return fmt.Errorf("Slack API error: %v", response["error"])
	}

	fmt.Println("📌 Message pinned successfully!")
	return nil
}

func sendSlackMessage(channelID string, ticket JiraIssue, caseURL string) error {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		return fmt.Errorf("Slack token not set")
	}
	slackURL := "https://slack.com/api/chat.postMessage"
	message := fmt.Sprintf(
		"🥷 Escalation-Ninja welcomes you in this Escalation Channel!!!\n\n\n"+
			"🔗 *Escalation for:*    <%s|%s>\n\n"+
			"🚨 *Priority:*         `%s`\n"+
			"🏷️ *Type:*             `%s`\n"+
			"📊 *Status:*           `%s`\n"+
			"📝 *Summary:* %s",
		caseURL, ticket.Key, ticket.Fields.Priority.Name, ticket.Fields.IssueType.Name, ticket.Fields.Status.Name, ticket.Fields.Summary,
	)

	msg := SlackMessage{
		Channel: channelID,
		Text:    message,
	}

	jsonBody, _ := json.Marshal(msg)

	req, err := http.NewRequest("POST", slackURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body) // Read the response body

	fmt.Println("\nDEBUG - Slack API Response:", string(body)) // Log response from Slack

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned non-200 status: %s", resp.Status)
	}

	var response SlackMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if !response.Ok {
		return fmt.Errorf("Slack API returned an error")
	}

	err = pinSlackMessage(channelID, response.Ts)
	if err != nil {
		return fmt.Errorf("Failed to pin message: %v", err)
	}

	return nil

}

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

	//after joining the channel, I'll send the trial message. If that works it will only be a matter of formatting it to rich text and all
	//sendSlackMessage(channelID)

	return nil
}

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
		return fmt.Errorf("no valid users to invite")
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
