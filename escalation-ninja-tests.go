/*package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin) //scanner will handle the user input

	fmt.Print("Write your command below: \n")

	//now I'll read the input from the user
	scanner.Scan()
	userInput := scanner.Text()

	//breaking userInput down into a words Array, so that I can check if the command is correctly launche the rest of the program
	words := strings.Fields(userInput)

	if len(words) <= 0 {
		//if the user only hits "Enter", this should prompt for the request again... for now... the Bot won't just trigger, when put in Slack :D
		fmt.Print("❌ empty command, please try again\n")
		main()
	} else if len(words) > 0 && (words[0] == "/ninjaescal" || words[0] == "/ninjaescalate") {
		fmt.Println("✅ Command recognized - will escalate in a second!")

		//once I made sure the first word is in fact `/ninjaescal` or `/ninjaescalate` I'll move to the next checks.
		if len(words) > 1 {
			//confirming the arguments!
			commandArgs := strings.Join(words[1:], " ")
			fmt.Println("Command Arguments: ", commandArgs)
		}
	} else {
		fmt.Println("❌ Invalid Command - please start with /ninjaescal or /ninjaescalate")
		main() //will repeat, for the sake of testing the app - will double check it this can work on a Slack app
	}
}
*/