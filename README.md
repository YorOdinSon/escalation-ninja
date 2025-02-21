# 🥷 Escalation-Ninja 🥷

A **Go-powered Slack bot** that automates the creation of escalation channels, ensuring that the right people are instantly invited for quick incident response.  

## 📌 Features  

✅ **Auto-creates Slack channels** for escalations  
✅ **Parses structured Slack commands**  
✅ **Automatically invites the command user** (and optionally, other team members)  
✅ **Uses Slack API (`users.list`) to resolve `@usernames` to real Slack IDs**  
✅ **No PII** stored—fully safe for public repositories 🎉  

---

## 🔧 Installation & Setup  

### 1️⃣ **Clone the Repo**  

```bash
git clone https://github.com/YOUR-USERNAME/escalation-ninja.git
cd escalation-ninja
```

### 2️⃣ **Set Up Your Slack App**

Go to the Slack API Portal and create a new app
Enable the following Bot Token Scopes under OAuth & Permissions:

* commands
* channels:manage
* channels:write.invites
* channels:join
* groups:write
* groups:write.invites
* users:read

Install the app in your workspace and grab the Bot User OAuth Token (xoxb-...)

### 3️⃣ **Set Up Environment Variables**

Create a .env file or export variables manually:

```bash
export SLACK_BOT_TOKEN="xoxb-your-slack-bot-token"
```
(For local development, you can use a .env file and load it using **godotenv**).

### 4️⃣ Run the Bot Locally

```bash
go run main.go
```
Your bot should now be listening for Slack commands!

### 🚀 Usage
Slack Command Format:

```bash
/ninjaescal case: <case-url> [invite: <@slack-tag> ...]
```

#### Example:

```bash
/ninjaescal case: https://jira.company.com/browse/TICKET-1234 invite: @alice @bob
```
✅ invite: is optional! If omitted, only the command user will be invited. 🥷

### 📦 Deployment

Set SLACK_BOT_TOKEN as an environment variable

### 🤝 Contributing

Pull requests are welcome! If you find a bug or want to suggest a feature, open an issue.

### 🛡️ Security & Best Practices

DO NOT hardcode secrets (SLACK_BOT_TOKEN). Use environment variables.
DO NOT expose API keys in logs or public repositories.

### 🎉 Credits

Built by @YorOdinSon

Want to chat? Tweet me at @FrancavillaYor

### 📜 License
MIT License – Use, modify, and share freely! 🚀